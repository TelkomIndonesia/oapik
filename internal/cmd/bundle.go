package cmd

import (
	"log"
	"os"

	"github.com/telkomindonesia/oapik/internal/bundle"
)

type Bundle struct {
	Input  string `arg:"input"`
	Output string `arg:"output" optional:"" `
}

func (b Bundle) Run(ctx Context) error {
	bytes, err := bundle.File(b.Input)
	if err != nil {
		log.Fatalln("fail to bundle file:", err)
	}

	bytes = append([]byte("# Code generated by openapi-utils. DO NOT EDIT.\n"), bytes...)
	switch b.Output {
	case "":
		if _, err := os.Stdout.Write(bytes); err != nil {
			log.Fatalln("fail to write stdout: ", err)
		}
	default:
		if err := os.WriteFile(b.Output, bytes, 0644); err != nil {
			log.Fatalln("fail to write file: ", err)
		}
	}
	return nil
}