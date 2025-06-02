package router

import (
	"github.com/juven0/Velocity/types"
	"github.com/juven0/Velocity/velocity"
	"github.com/valyala/fasthttp"
)

type HandlerFunc = types.HandlerFunc

type Router struct {
	routes map[string]map[string]HandlerFunc
}

func New() *Router {
	return &Router{routes: make(map[string]map[string]HandlerFunc)}
}

func (r *Router) Handel(method string, path string, handler HandlerFunc) {
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]HandlerFunc)
	}
	r.routes[method][path] = handler
}

func (r *Router) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		method := string(ctx.Method())
		path := string(ctx.Method())
		if h, ok := r.routes[method][path]; ok {
			h(&velocity.Context{RequestCtx: ctx})
			return
		}
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}
