package main

import (
	"github.com/juven0/Velocity/types"
	"github.com/juven0/Velocity/velocity"
)

func main() {
	app := velocity.New()

	app.Get("/demo", func(ctx *types.Context) error {
		return ctx.Text("GET /demo")
	})

	app.Get("/demo/:id", func(ctx *types.Context) error {
		id := ctx.Param("id")
		return ctx.Text("GET /demo/" + id)
	})

	app.Post("/demo", func(ctx *types.Context) error {
		name := ctx.FormValue("name")
		return ctx.Text("POST /demo with name: " + name)
	})

	app.Put("/demo/:id", func(ctx *types.Context) error {
		id := ctx.Param("id")
		value := ctx.FormValue("value")
		return ctx.Text("PUT /demo/" + id + " with value: " + value)
	})

	app.Delete("/demo/:id", func(ctx *types.Context) error {
		id := ctx.Param("id")
		return ctx.Text("DELETE /demo/" + id)
	})

	app.Listen(":8000")
}
