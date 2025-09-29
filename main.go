package main

import (
	"context"
	"errors"
	"log"

	"github.com/jylitalo/expenses/cmd"
	"github.com/jylitalo/expenses/config"
)

func main() {
	ctx, errConfig := config.Read(context.Background())
	if err := errors.Join(errConfig, cmd.Execute(ctx)); err != nil {
		log.Fatal(err) //nolint:gocritic
	}
}
