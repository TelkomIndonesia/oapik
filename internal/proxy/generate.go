package proxy

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/telkomindonesia/oapik/internal/util"
	"golang.org/x/tools/imports"
)

const prefixUpstream = "upstream-"

func addTemplateFunc(pe ProxyExtension) {
	codegen.TemplateFunctions["upstreamOperationID"] = func(opid string) string {
		for k, v := range pe.Proxied() {
			if opid != k.OperationId {
				continue
			}

			uop, _ := v.GetUpstreamOperation()
			return codegen.ToCamelCase(prefixUpstream + uop.OperationId)
		}
		return ""
	}
	codegen.TemplateFunctions["upstream"] = func(opid string) string {
		for k, v := range pe.Proxied() {
			if opid != codegen.ToCamelCase(k.OperationId) {
				continue
			}
			return codegen.ToCamelCase(v.GetName())
		}
		return ""
	}
	codegen.TemplateFunctions["upstreams"] = func() (a []string) {
		for _, p := range pe.Upstream() {
			a = append(a, codegen.ToCamelCase(p.GetName()))
		}
		return
	}
}

type GenerateOptions struct {
	PackageName string
}

func Generate(ctx context.Context, specPath string, opts GenerateOptions) (bytes []byte, err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return nil, fmt.Errorf("fail to create proxy extension: %w", err)
	}
	addTemplateFunc(pe)

	{
		spec, _, _, err := pe.CreateProxyDoc()
		if err != nil {
			return nil, fmt.Errorf("fail to create proxy doc: %w", err)
		}
		kinspec, err := loadKinDoc(spec)
		if err != nil {
			return nil, fmt.Errorf("fail to reload proxy doc with kin: %w", err)
		}

		t, err := loadTemplates("proxy")
		if err != nil {
			return nil, fmt.Errorf("fail to load template: %w", err)
		}

		code, err := codegen.Generate(kinspec, codegen.Configuration{
			PackageName: opts.PackageName,
			Compatibility: codegen.CompatibilityOptions{
				AlwaysPrefixEnumValues: true,
			},
			Generate: codegen.GenerateOptions{
				EchoServer: true,
				Strict:     true,
				Models:     true,
			},
			OutputOptions: codegen.OutputOptions{
				UserTemplates: t,
				SkipFmt:       true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("fail to generate code: %w", err)
		}
		bytes = append(bytes, []byte(code)...)
	}

	{
		t, err := loadTemplates("upstream")
		if err != nil {
			return nil, fmt.Errorf("fail to load template: %w", err)
		}

		generated := map[*libopenapi.DocumentModel[v3.Document]]struct{}{}
		for _, pop := range pe.Proxied() {
			doc, err := pop.GetOpenAPIDoc()
			if err != nil {
				return nil, fmt.Errorf("fail to find upstream openapi doc: %w", err)
			}
			docv3, _ := doc.BuildV3Model()
			if _, ok := generated[docv3]; ok {
				continue
			}

			// add prefix
			for m := range orderedmap.Iterate(ctx, docv3.Model.Paths.PathItems) {
				for _, op := range util.GetOperationsMap(m.Value()) {
					op.OperationId = prefixUpstream + op.OperationId
				}
			}
			components := util.NewComponents()
			components.CopyComponents(docv3, "")
			components.CopyComponents(docv3, prefixUpstream)
			_, _, ndocv3, _ := components.RenderAndReloadWith(doc)
			components = util.NewComponents()
			components.CopyAndLocalizeComponents(ndocv3, prefixUpstream)
			spec, _ := components.RenderWith(ndocv3)

			kinspec, err := loadKinDoc(spec)
			if err != nil {
				return nil, fmt.Errorf("fail to reload proxy doc with kin: %w", err)
			}

			code, err := codegen.Generate(kinspec, codegen.Configuration{
				Compatibility: codegen.CompatibilityOptions{
					AlwaysPrefixEnumValues: true,
				},
				Generate: codegen.GenerateOptions{
					EchoServer: true,
					Strict:     true,
					Models:     true,
				},
				OutputOptions: codegen.OutputOptions{
					UserTemplates: t,
					SkipFmt:       true,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("fail to generate code: %w", err)
			}
			bytes = append(bytes, []byte(code)...)

			generated[docv3] = struct{}{}
		}
	}

	bytes, err = imports.Process("oapi.go", bytes, nil)
	if err != nil {
		return nil, fmt.Errorf("error formatting Go code: %w", err)
	}
	return
}

func loadKinDoc(data []byte) (doc *openapi3.T, err error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err = loader.LoadFromData(data)
	return
}

//go:embed templates/*
var templates embed.FS

func loadTemplates(dir string) (t map[string]string, err error) {
	t = make(map[string]string)
	err = fs.WalkDir(templates, path.Join("templates", dir), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		buf, err := templates.ReadFile(p)
		if err != nil {
			return fmt.Errorf("error reading file '%s': %w", p, err)
		}

		templateName := strings.TrimPrefix(p, path.Join("templates", dir)+"/")
		t[templateName] = string(buf)
		return nil
	})
	return

}
