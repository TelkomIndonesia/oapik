package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/oapik/internal/util"
)

const xProxyExtensionName = "x-proxy"

type ProxyExtension struct {
	specPath string
	specDir  string

	doc      libopenapi.Document
	docv3    *libopenapi.DocumentModel[v3.Document]
	proxied  map[*v3.Operation]*ProxyOperation
	upstream map[string]*Proxy
}

func NewProxyExtension(ctx context.Context, specPath string) (pe ProxyExtension, err error) {
	pe.specPath = specPath
	pe.specDir, err = filepath.Abs(filepath.Dir(specPath))
	if err != nil {
		return pe, fmt.Errorf("fail to determine spec file base directory: %w", err)
	}

	if err = pe.loadDoc(); err != nil {
		return
	}
	if err = pe.loadProxy(ctx); err != nil {
		return
	}
	if err = pe.pruneAndPrefixUpstream(ctx); err != nil {
		return
	}
	if err = pe.compile(); err != nil {
		return
	}

	return
}

func (pe *ProxyExtension) loadDoc() (err error) {
	specBytes, err := os.ReadFile(pe.specPath)
	if err != nil {
		return fmt.Errorf("fail to read spec file: %w", err)
	}

	doc, err := libopenapi.NewDocument([]byte(specBytes))
	if err != nil {
		return fmt.Errorf("failed to create openapi document: %w", err)
	}
	docv3, errs := doc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return fmt.Errorf("failed to create openapi v3 document: %w", err)
	}

	pe.doc, pe.docv3 = doc, docv3
	return
}

func (pe *ProxyExtension) loadProxy(ctx context.Context) (err error) {
	pe.upstream = map[string]*Proxy{}
	if pe.docv3.Model.Components.Extensions != nil {
		ex, ok := pe.docv3.Model.Components.Extensions.Get(xProxyExtensionName)
		if ok {
			if err = ex.Decode(pe.upstream); err != nil {
				return fmt.Errorf("fail to decode `%s` component :%w", xProxyExtensionName, err)
			}

			for k, v := range pe.upstream {
				v.Name = k
				v.Spec = path.Join(pe.specDir, v.Spec)
			}
		}
	}

	pe.proxied = map[*v3.Operation]*ProxyOperation{}
	for m := range orderedmap.Iterate(ctx, pe.docv3.Model.Paths.PathItems) {
		for method, op := range util.GetOperationsMap(m.Value()) {
			if op.Extensions == nil {
				continue
			}
			ex, ok := op.Extensions.Get(xProxyExtensionName)
			if !ok {
				continue
			}

			var pop ProxyOperation
			if err = ex.Decode(&pop); err != nil {
				return fmt.Errorf("fail to decode Proxy Operation : %w", err)
			}
			if pop.Spec == "" && pop.Proxy != nil && pop.Proxy.Name != "" {
				pop.Proxy, ok = pe.upstream[pop.Name]
				if !ok {
					return fmt.Errorf("invalid proxy definition for %s: no spec is provided", pop.Proxy.Name)
				}
			} else {
				pop.Spec = path.Join(pe.specDir, pop.Spec)
			}
			if pop.Path == "" {
				pop.Path = m.Key()
			}
			if pop.Method == "" {
				pop.Method = method
			}

			_, err = pop.GetOpenAPIDoc()
			if err != nil {
				return fmt.Errorf("fail to load upstream openapi spec: %w", err)
			}
			_, err = pop.GetUpstreamOperation()
			if err != nil {
				return fmt.Errorf("fail to find upstream operation: %w", err)
			}
			_, err = pop.GetProxiedParameters()
			if err != nil {
				return fmt.Errorf("fail to get proxied parameter: %w", err)
			}

			pe.proxied[op] = &pop
		}
	}
	return
}

func (pe *ProxyExtension) pruneAndPrefixUpstream(ctx context.Context) (err error) {
	upstreams := map[libopenapi.Document]map[*v3.Operation]map[*ProxyOperation]struct{}{}
	for _, pop := range pe.proxied {
		doc, _ := pop.GetOpenAPIDoc()
		uop, _ := pop.GetUpstreamOperation()
		if _, ok := upstreams[doc]; !ok {
			upstreams[doc] = map[*v3.Operation]map[*ProxyOperation]struct{}{}
		}
		if _, ok := upstreams[doc][uop]; !ok {
			upstreams[doc][uop] = map[*ProxyOperation]struct{}{}
		}
		upstreams[doc][uop][pop] = struct{}{}
	}

	for doc, uopPopMap := range upstreams {
		docv3, _ := doc.BuildV3Model()
		prefix := util.MapFirstEntry(util.MapFirstEntry(uopPopMap).Value).Key.GetName() + "-"
		prefix = codegen.ToCamelCaseWithInitialisms(prefix)
		prefixer := appendPrefix(prefix)

		// add prefix to operation id
		opmap := map[*v3.Operation]struct{}{}
		for uop := range uopPopMap {
			opmap[uop] = struct{}{}
			opID := uop.OperationId
			if opID != "" {
				opID = strings.ToUpper(opID[:1]) + opID[1:]
			}
			uop.OperationId = prefix + opID
		}

		// delete unused operations and path items
		unusedPathItem := map[string]struct{}{}
		for m := range orderedmap.Iterate(ctx, docv3.Model.Paths.PathItems) {
			pathItem := m.Value()
			for method, op := range util.GetOperationsMap(pathItem) {
				if _, ok := opmap[op]; ok {
					continue
				}
				util.SetOperation(pathItem, method, nil)
			}
			if len(util.GetOperationsMap(pathItem)) == 0 {
				unusedPathItem[m.Key()] = struct{}{}
			}
		}
		for pi := range unusedPathItem {
			docv3.Model.Paths.PathItems.Delete(pi)
		}

		// recreate the doc so that we could get references of used operations only
		// also add components with prefix so that it doesn't trigger error log from libopenapi
		components := util.NewComponents()
		err := components.CopyComponents(docv3, nil)
		if err != nil {
			return fmt.Errorf("fail to copy components: %w", err)
		}
		err = components.CopyComponents(docv3, prefixer)
		if err != nil {
			return fmt.Errorf("fail to copy components with prefix: %w", err)
		}
		_, doc, docv3, err = components.RenderAndReloadWith(doc)
		if err != nil {
			return fmt.Errorf("fail to render and reload upstream doc: %w", err)
		}

		// rerender with prefixed added to all components
		components = util.NewComponents()
		err = components.CopyAndLocalizeComponents(docv3, prefixer)
		if err != nil {
			return fmt.Errorf("fail to copy components with prefix: %w", err)
		}
		_, doc, docv3, err = components.RenderAndReloadWith(doc)
		if err != nil {
			return fmt.Errorf("fail to render and reload upstream doc: %w", err)
		}

		// store it
		for _, popmap := range uopPopMap {
			for pop := range popmap {
				*pop = pop.WithReloadedDoc(doc)
			}
		}
	}
	return
}

// compile proxy document
func (pe *ProxyExtension) compile() (err error) {
	for op, pop := range pe.proxied {
		uop, _ := pop.GetUpstreamOperation()
		params, _ := pop.GetProxiedParameters()

		// copy operation
		opParam := util.CopyParameters(op.Parameters, params...)
		opID := op.OperationId
		opSecurity := op.Security
		opExt := op.Extensions
		opExt.Delete(xProxyExtensionName)

		*op = *uop

		op.Parameters = opParam
		if opID != "" {
			op.OperationId = opID
		}
		op.Security = opSecurity
		for m := range orderedmap.Iterate(context.Background(), op.Extensions) {
			opExt.Set(m.Key(), m.Value())
		}
		op.Extensions = opExt
	}

	return
}

func (pe *ProxyExtension) Proxied() map[*v3.Operation]*ProxyOperation {
	return pe.proxied
}

func (pe *ProxyExtension) Upstream() map[string]*Proxy {
	return pe.upstream
}

func (pe *ProxyExtension) CreateProxyDoc() (b []byte, ndoc libopenapi.Document, docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	components := util.NewComponents()

	copied := map[*libopenapi.DocumentModel[v3.Document]]struct{}{}
	for _, pop := range pe.proxied {
		docv3, _ := pop.GetOpenAPIV3Doc()
		if _, ok := copied[docv3]; ok {
			continue
		}

		err := components.CopyComponents(docv3, nil)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to copy localized components: %w", err)
		}

		copied[docv3] = struct{}{}
	}

	err = components.CopyComponents(pe.docv3, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to copy components on proxy doc: %w", err)
	}

	components.Extensions.Delete(xProxyExtensionName)

	return components.RenderAndReloadWith(pe.doc)
}

func appendPrefix(prefix string) func(name string) string {
	return func(name string) string {
		if name == "" {
			return prefix
		}
		return prefix + strings.ToUpper(name[:1]) + name[1:]
	}
}
