package main

import (
	"errors"
	"os"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestBundle(t *testing.T) {
	src := "./testdata/profile/profile.yml"
	bytes, err := bundleFile(src)
	require.NoError(t, err)
	err = os.WriteFile("testoutput/profile.yml", []byte(bytes), 0o644)
	require.NoError(t, err)
	doc, err := libopenapi.NewDocument(bytes)
	require.NoError(t, err)
	_, errs := doc.BuildV3Model()
	require.NoError(t, errors.Join(errs...))
}
