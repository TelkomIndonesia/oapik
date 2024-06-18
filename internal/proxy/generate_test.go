package proxy

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	bytes, err := Generate(context.Background(), "./testdata/spec-proxy.yml", GenerateOptions{
		PackageName: "testoutput",
	})
	require.NoError(t, err)
	err = os.WriteFile("testoutput/oapi-proxy.go", []byte(bytes), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("go", "test", ".", "-v")
	cmd.Dir = "testoutput"
	out, err := cmd.Output()
	t.Log("\n" + string(out))
	require.NoError(t, err, "test for generated code should not return error")
	assert.Equal(t, 0, cmd.ProcessState.ExitCode(), "test for generated code should not exit with non-zero status code")
}
