package cmd

import (
	"context"

	"github.com/alecthomas/kong"
)

type Context struct {
	context.Context
}

type CLI struct {
	Bundle Bundle `cmd:"bundle"`
	Proxy  Bundle `cmd:"proxy"`
}

type CMD struct{}

func (c *CMD) Start(ctx context.Context) (err error) {
	cmd := CLI{}
	k := kong.Parse(&cmd)
	return k.Run(Context{ctx})
}

func New() *CMD {
	return &CMD{}
}
