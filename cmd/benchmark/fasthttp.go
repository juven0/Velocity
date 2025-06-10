package benchmark

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

func simpleHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain")
	ctx.SetStatusCode(fasthttp.StatusOK)
	fmt.Fprint(ctx, "Hello, fasthttp!")
}

func main() {
	fmt.Println("🚀 Server running on http://localhost:8080")
	if err := fasthttp.ListenAndServe(":8080", simpleHandler); err != nil {
		panic("Error starting server: " + err.Error())
	}
}
