package proxy

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestCompile(t *testing.T) {
	src := "./testdata/spec-proxy.yml"
	bytes, err := Compile(context.Background(), src)
	require.NoError(t, err)
	err = os.WriteFile("testoutput/oapi-proxy.yml", []byte(bytes), 0o644)
	require.NoError(t, err)
	doc, err := libopenapi.NewDocument(bytes)
	require.NoError(t, err)
	_, errs := doc.BuildV3Model()
	require.NoError(t, errors.Join(errs...))
}
