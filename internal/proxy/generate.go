package proxy

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
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
	"github.com/twpayne/go-jsonstruct/v3"
	"golang.org/x/tools/imports"
)

var prefixer = appendPrefix("upstream-")

var codegenNameNormalizerFunctionName = codegen.NameNormalizerFunctionUnset
var codegenNameNormalizerFunction = codegen.NameNormalizers[codegenNameNormalizerFunctionName]

func addTemplateFunc(pe ProxyExtension) {
	codegen.TemplateFunctions["upstreamOperationID"] = func(opid string) string {
		for k, v := range pe.Proxied() {
			if opid != codegenNameNormalizerFunction(k.OperationId) {
				continue
			}

			uop, _ := v.GetUpstreamOperation()
			return prefixer(uop.OperationId)
		}
		return ""
	}
	codegen.TemplateFunctions["upstream"] = func(opid string) string {
		for k, v := range pe.Proxied() {
			if opid != codegenNameNormalizerFunction(k.OperationId) {
				continue
			}
			return codegen.ToCamelCase(v.GetName())
		}
		return ""
	}
	codegen.TemplateFunctions["upstreams"] = func() (a []string) {
		for _, p := range pe.Upstream() {
			a = append(a, codegenNameNormalizerFunction(p.GetName()))
		}
		return
	}
	codegen.TemplateFunctions["writeExtensionType"] = func(ops []codegen.OperationDefinition, name string) (string, error) {
		jsGenerator := jsonstruct.NewGenerator(
			jsonstruct.WithPackageName("nopackage"),
			jsonstruct.WithTypeName(name),
		)
		for _, op := range ops {
			m := map[string]interface{}{}
			for k, v := range op.Spec.Extensions {
				m[strings.TrimPrefix(k, "x-")] = v
			}
			jsGenerator.ObserveValue(m)
		}
		b, err := jsGenerator.Generate()
		if err != nil {
			return "", fmt.Errorf("fail to generate type definition %w", err)
		}
		b = bytes.Replace(b, []byte("package nopackage\n"), []byte{}, 1)
		return string(b), err
	}

	codegen.TemplateFunctions["writeExtensionData"] = func(op codegen.OperationDefinition) (string, error) {
		m := map[string]interface{}{}
		for k, v := range op.Spec.Extensions {
			m[strings.TrimPrefix(k, "x-")] = v
		}

		b, err := json.Marshal(m)
		if err != nil {
			return "", fmt.Errorf("fail to marshall to json: %w", err)
		}
		b, err = json.Marshal(string(b))

		return string(b), err
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
				UserTemplates:  t,
				SkipFmt:        true,
				NameNormalizer: string(codegenNameNormalizerFunctionName),
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
					op.OperationId = prefixer(op.OperationId)
				}
			}
			components := util.NewComponents()
			components.CopyComponents(docv3, nil)
			components.CopyComponents(docv3, prefixer)
			_, _, ndocv3, _ := components.RenderAndReloadWith(doc)
			components = util.NewComponents()
			components.CopyAndLocalizeComponents(ndocv3, prefixer)
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
					UserTemplates:  t,
					SkipFmt:        true,
					NameNormalizer: string(codegenNameNormalizerFunctionName),
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
