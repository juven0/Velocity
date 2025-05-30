package velocity

import (
	"github.com/juven0/Velocity/test/router"
	"github.com/valyala/fasthttp"
)

type App struct {
	router      *router.Router
	middlewares []HandlerFunc
}

func New() *App {
	return &App{
		router:      router.New(),
		middlewares: []HandlerFunc{},
	}
}

func (a *App) Use(mw HandlerFunc) {
	a.middlewares = append(a.middlewares, mw)
}

func (a *App) Get(path string, handler HandlerFunc) {
	a.router.Handle("GET", path, a.chain(handler))
}

func (a *App) chain(final HandlerFunc) HandlerFunc {
	return func(ctx *Context) error {
		h := final
		for i := len(a.middlewares) - 1; i >= 0; i-- {
			next := h
			mw := a.middlewares[i]
			h = func(c *Context) error {
				return mw(c.WithNext(next))
			}
		}
		return h(ctx)
	}
}

func (a *App) Listen(addr string) error {
	return fasthttp.ListenAndServe(addr, a.router.Handler())
}
