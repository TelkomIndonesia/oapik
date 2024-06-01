package proxy

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	baselow "github.com/pb33f/libopenapi/datamodel/low/base"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/openapi-utils/cmd/proxy/internal/proxy/config"
)

func CompileByte(ctx context.Context, specBytes []byte, specDir string) (newspec []byte, doc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	proxyDoc, err := libopenapi.NewDocument([]byte(specBytes))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create openapi document: %w", err)
	}
	proxyDocv3, errs := proxyDoc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create openapi v3 document: %w", err)
	}
	initComponents(proxyDocv3)

	// build the proxy
	proxies := map[string]*config.Proxy{}
	if proxyDocv3.Model.Components.Extensions != nil {
		ex, ok := proxyDocv3.Model.Components.Extensions.Get("x-proxy")
		if ok {
			if err = ex.Decode(proxies); err != nil {
				return nil, nil, nil, fmt.Errorf("fail to decode `x-proxy` component :%w", err)
			}
			for k, v := range proxies {
				v.Name = &k
				v.Spec = path.Join(specDir, v.Spec)
			}
		}
	}
	proxyOperations := map[*config.ProxyOperation]*v3.Operation{}
	upstreamDocsOri := map[libopenapi.Document]map[*config.ProxyOperation]*v3.Operation{}
	for m := range orderedmap.Iterate(ctx, proxyDocv3.Model.Paths.PathItems) {
		for _, op := range getOperationsMap(m.Value()) {
			if op.Extensions == nil {
				continue
			}

			ex, ok := op.Extensions.Get("x-proxy")
			if !ok {
				continue
			}

			var pop config.ProxyOperation
			err = ex.Decode(&pop)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fail to decode Proxy Operation : %w", err)
			}

			if pop.Spec == "" && pop.Proxy != nil && pop.Proxy.Name != nil {
				pop.Proxy, ok = proxies[*pop.Name]
				if !ok {
					return nil, nil, nil, fmt.Errorf("invalid proxy definition for %s: no spec is provided", *pop.Proxy.Name)
				}
			} else {
				pop.Spec = path.Join(specDir, pop.Spec)
			}

			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fail to load `x-proxy` :%w", err)
			}
			if _, ok := upstreamDocsOri[doc]; !ok {
				upstreamDocsOri[doc] = map[*config.ProxyOperation]*v3.Operation{}
			}
			docv3, _ := doc.BuildV3Model()
			val, ok := docv3.Model.Paths.PathItems.Get(pop.Path)
			if !ok {
				continue
			}
			uop := getOperation(val, pop.Method)
			if uop == nil {
				continue
			}

			proxyOperations[&pop] = op
			upstreamDocsOri[doc][&pop] = uop
		}
	}

	// rerender upstream document to contains only used operation and duplicated prefixed schema
	upstreamDocs := map[libopenapi.Document]string{}
	proxyOperationUpstreamDocs := map[*config.ProxyOperation]libopenapi.Document{}
	for doc, popmap := range upstreamDocsOri {
		docV3, _ := doc.BuildV3Model()
		var docName string
		for pop := range popmap {
			docName = pop.GetName()
		}

		// delete unused operation
		opmap := map[*v3.Operation]struct{}{}
		for _, v := range popmap {
			opmap[v] = struct{}{}
		}
		for m := range orderedmap.Iterate(ctx, docV3.Model.Paths.PathItems) {
			pathItem := m.Value()
			for method, op := range getOperationsMap(m.Value()) {
				if _, ok := opmap[op]; ok {
					continue
				}
				setOperation(pathItem, method, nil)
			}
		}

		// copy schema and add prefix
		for _, ref := range docV3.Index.GetRawReferencesSequenced() {
			if !strings.HasPrefix(ref.Definition, "#/components/schemas") {
				continue
			}

			strings.HasPrefix(ref.Definition, "#/components/schemas")
			schemaProxy, err := baselow.ExtractSchema(ctx, ref.Node, ref.Index)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fail to recreate schema: %w", err)
			}

			docV3.Model.Components.Schemas.Set(docName+ref.Name, base.NewSchemaProxy(schemaProxy))
		}

		// rerender
		_, doc, docV3, errs = doc.RenderAndReload()
		if err := errors.Join(errs...); err != nil {
			return nil, nil, nil, fmt.Errorf("faill to render and reload openapi doc: %w", err)
		}

		// store the result
		upstreamDocs[doc] = docName
		for pop := range popmap {
			proxyOperationUpstreamDocs[pop] = doc
		}
	}

	// attach prefix to all components and copy them to proxy document
	for doc, docName := range upstreamDocs {
		docV3, _ := doc.BuildV3Model()
		for _, ref := range docV3.Index.GetRawReferencesSequenced() {
			switch {
			case strings.HasPrefix(ref.Definition, "#/components/schemas"):
				schemaProxy, err := baselow.ExtractSchema(ctx, ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to recreate schema: %w", err)
				}

				name := docName + ref.Name
				refname := "#/components/schemas/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Schemas.Set(name, base.NewSchemaProxy(schemaProxy))

			case strings.HasPrefix(ref.Definition, "#/components/parameters"):
				v, err := low.ExtractObject[*v3low.Parameter](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract paramater: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/parameters/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Parameters.Set(name, v3.NewParameter(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/requestBodies"):
				v, err := low.ExtractObject[*v3low.RequestBody](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract paramater: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/requestBodies/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.RequestBodies.Set(name, v3.NewRequestBody(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/headers"):
				v, err := low.ExtractObject[*v3low.Header](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract paramater: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/headers/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Headers.Set(name, v3.NewHeader(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/responses"):
				v, err := low.ExtractObject[*v3low.Response](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract response: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/responses/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Responses.Set(name, v3.NewResponse(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/securitySchemes"):
				v, err := low.ExtractObject[*v3low.SecurityScheme](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract response: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/securitySchemes/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.SecuritySchemes.Set(name, v3.NewSecurityScheme(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/examples"):
				v, err := low.ExtractObject[*baselow.Example](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract response: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/examples/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Examples.Set(name, base.NewExample(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/links"):
				v, err := low.ExtractObject[*v3low.Link](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract response: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/links/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Links.Set(name, v3.NewLink(v.Value))

			case strings.HasPrefix(ref.Definition, "#/components/callbacks"):
				v, err := low.ExtractObject[*v3low.Callback](ctx, "", ref.Node, ref.Index)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("fail to extract response: %w", err)
				}
				v.Value.Build(ctx, v.KeyNode, v.ValueNode, ref.Index)

				name := docName + ref.Name
				refname := "#/components/links/" + name
				ref.Node.Content = base.CreateSchemaProxyRef(refname).GetReferenceNode().Content
				proxyDocv3.Model.Components.Callbacks.Set(name, v3.NewCallback(v.Value))
			}
		}
	}

	// compile proxy document
	for pop, op := range proxyOperations {
		ud, ok := proxyOperationUpstreamDocs[pop]
		if !ok {
			continue
		}

		// find upstream path operation, with parameter inherited from upstream path
		udv3, _ := ud.BuildV3Model()
		up, ok := udv3.Model.Paths.PathItems.Get(pop.Path)
		if !ok {
			continue
		}
		uop := getOperation(up, pop.Method)
		if uop == nil {
			continue
		}
		var uParams []*v3.Parameter
		injectedParamMap := map[parameterKey]struct{}{}
		for _, p := range pop.Inject.Parameters {
			injectedParamMap[parameterKey{name: p.Name, in: p.In}] = struct{}{}
		}
		for _, p := range copyParameters(up.Parameters, uop.Parameters...) {
			if _, ok := injectedParamMap[parameterKey{name: p.Name, in: p.In}]; ok {
				continue
			}
			uParams = append(uParams, p)
		}

		// copy operation
		opParam := copyParameters(op.Parameters, uParams...)
		opID := op.OperationId
		opSecurity := op.Security
		*op = *uop
		op.Parameters = opParam
		op.OperationId = opID
		op.Security = opSecurity
	}
	by, proxyDoc, proxyDocv3, errs := proxyDoc.RenderAndReload()

	return by, proxyDoc, proxyDocv3, errors.Join(errs...)
}