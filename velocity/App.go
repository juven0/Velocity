package velocity

import (
	"log"
	"time"

	"github.com/juven0/Velocity/internal/router"
	"github.com/juven0/Velocity/types"

	"github.com/valyala/fasthttp"
)

type App struct {
	router      *router.Router
	middlewares []types.HandlerFunc
	config      *ServerConfig
}

type ServerConfig struct {
	ReadBufferSize                int
	WriteBufferSize               int
	MaxRequestBodySize            int
	Concurrency                   int
	DisableKeepalive              bool
	DisableHeaderNamesNormalizing bool
	ReadTimeout                   time.Duration
	WriteTimeout                  time.Duration
	IdleTimeout                   time.Duration
}

func New() *App {
	return &App{
		router:      router.New(),
		middlewares: []types.HandlerFunc{},
	}
}

func defaultConfig() *ServerConfig {
	return &ServerConfig{
		ReadBufferSize:                4096,
		WriteBufferSize:               4096,
		MaxRequestBodySize:            1024 * 1024,
		Concurrency:                   256 * 1024,
		DisableKeepalive:              false,
		DisableHeaderNamesNormalizing: true,
		ReadTimeout:                   10 * time.Second,
		WriteTimeout:                  10 * time.Second,
		IdleTimeout:                   120 * time.Second,
	}
}

func (a *App) Config() *ServerConfig {
	return a.config
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
	if len(a.middlewares) == 0 {
		return final
	}

	return func(ctx *types.Context) error {
		var index int

		var next func() error
		next = func() error {
			if index >= len(a.middlewares) {
				return final(ctx)
			}
			middleware := a.middlewares[index]
			index++

			ctxWithNext := ctx.WithNext(func(*types.Context) error {
				return next()
			})

			return middleware(ctxWithNext)
		}

		return next()
	}
}

func (a *App) Handler() fasthttp.RequestHandler {
	routerHandler := a.router.Handler()

	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.SetServer("Velocity")

		routerHandler(ctx)
	}
}

func (a *App) Listen(addr string) error {
	printBanner(addr)

	server := &fasthttp.Server{
		Handler:                       a.Handler(),
		DisableKeepalive:              a.config.DisableKeepalive,
		ReadBufferSize:                a.config.ReadBufferSize,
		WriteBufferSize:               a.config.WriteBufferSize,
		MaxRequestBodySize:            a.config.MaxRequestBodySize,
		Concurrency:                   a.config.Concurrency,
		DisableHeaderNamesNormalizing: a.config.DisableHeaderNamesNormalizing,
		ReadTimeout:                   a.config.ReadTimeout,
		WriteTimeout:                  a.config.WriteTimeout,
		IdleTimeout:                   a.config.IdleTimeout,
		ReduceMemoryUsage:             false,
		TCPKeepalive:                  true,
	}

	log.Printf("Server starting on %s with %d max goroutines", addr, a.config.Concurrency)
	return server.ListenAndServe(addr)
}
