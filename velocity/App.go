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
	a.router.Handel("GET", path, a.chain(handler))
}

func (a *App) chain(final types.HandlerFunc) (types.HandlerFunc) {
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
	return fasthttp.ListenAndServe(addr, a.router.Handler())
}
