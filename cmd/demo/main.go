package main

import (
	"github.com/juven0/Velocity/types"
	"github.com/juven0/Velocity/velocity"
)

func main() {
	app := velocity.New()

	app.Get("/", func(ctx *types.Context) error {
		return ctx.Text("Hello, World!")
	})

	app.Listen(":8000")
}
