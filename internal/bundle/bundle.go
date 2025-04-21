package bundle

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/telkomindonesia/oapik/internal/util"
)

func File(path string) (bytes []byte, err error) {
	by, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fail to read file :%w", err)
	}
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(by), &datamodel.DocumentConfiguration{
		BasePath:                filepath.Dir(path),
		ExtractRefsSequentially: true,
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	})
	if err != nil {
		return nil, fmt.Errorf("fail to load openapi spec: %w", err)
	}

	bytes, err = bundle(doc)
	if err != nil {
		return nil, fmt.Errorf("fail to bundle: %w", err)
	}

	return
}

func bundle(doc libopenapi.Document) (bytes []byte, err error) {
	docv3, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("fail to re-build openapi spec: %w", errors.Join(errs...))
	}

	// create stub components and localize all references
	components := util.NewComponents()
	err = components.CopyAndLocalizeComponents(docv3, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to copy stub components: %w", err)
	}

	return components.RenderWith(docv3)
}
