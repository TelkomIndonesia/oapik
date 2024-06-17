package proxy

import (
	"context"
)

func Compile(ctx context.Context, specPath string) (newspec []byte, err error) {
	pe, err := NewProxyExtension(ctx, specPath)
	if err != nil {
		return nil, err
	}

	newspec, _, _, err = pe.CreateProxyDoc()
	return
}
