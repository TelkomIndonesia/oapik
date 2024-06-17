package main

import (
	"context"
	"log"

	"github.com/telkomindonesia/oapik/internal/cmd"
)

func main() {
	err := cmd.New().Start(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
