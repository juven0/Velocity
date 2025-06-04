package velocity

import (
	"github.com/juven0/Velocity/internal/router"
	"github.com/juven0/Velocity/types"

	"github.com/valyala/fasthttp"
)

type App struct {
	router      *router.Router
	middlewares []types.HandlerFunc
}

func New() *App {
	return &App{
		router:      router.New(),
		middlewares: []types.HandlerFunc{},
	}
}

func (a *App) Use(mw types.HandlerFunc) {
	a.middlewares = append(a.middlewares, mw)
}

func (a *App) Get(path string, handler types.HandlerFunc) {
	a.router.Handel("GET", path, handler)
}

func (a *App) Post(path string, handler types.HandlerFunc) {
	a.router.Handel("POST", path, handler)
}

func (a *App) Put(path string, handler types.HandlerFunc) {
	a.router.Handel("PUT", path, handler)
}

func (a *App) Delete(path string, handler types.HandlerFunc) {
	a.router.Handel("DELETE", path, handler)
}

func (a *App) Patch(path string, handler types.HandlerFunc) {
	a.router.Handel("PATCH", path, handler)
}

func (a *App) Options(path string, handler types.HandlerFunc) {
	a.router.Handel("OPTIONS", path, handler)
}

func (a *App) Head(path string, handler types.HandlerFunc) {
	a.router.Handel("HEAD", path, handler)
}

func (a *App) chain(final types.HandlerFunc) types.HandlerFunc {
	return func(ctx *types.Context) error {
		h := final
		for i := len(a.middlewares) - 1; i >= 0; i-- {
			next := h
			mw := a.middlewares[i]
			h = func(c *types.Context) error {
				return mw(c.WithNext(next))
			}
		}
		return h(ctx)
	}
}

func (a *App) Listen(addr string) error {
	printBanner(addr)
	return fasthttp.ListenAndServe(addr, a.router.Handler())
}
