package velocity

import (
	"log"
	"sync"
	"time"

	"github.com/juven0/Velocity/internal/router"
	"github.com/juven0/Velocity/types"

	"github.com/valyala/fasthttp"
)

var contextPool = sync.Pool{
	New: func() interface{} {
		return &types.Context{}
	},
}

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
		config:      defaultConfig(),
	}
}

// Configuration optimisée pour la performance
func defaultConfig() *ServerConfig {
	return &ServerConfig{
		ReadBufferSize:                16384,            
		WriteBufferSize:               16384,            
		MaxRequestBodySize:            10 * 1024 * 1024, 
		Concurrency:                   1024 * 1024,      
		DisableKeepalive:              false,
		DisableHeaderNamesNormalizing: true,
		ReadTimeout:                   5 * time.Second,  
		WriteTimeout:                  5 * time.Second,  
		IdleTimeout:                   60 * time.Second, 
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

// Chain middleware - optimisé pour éviter les allocations inutiles
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

	// Si pas de middleware, utiliser directement le router
	if len(a.middlewares) == 0 {
		return func(ctx *fasthttp.RequestCtx) {
			// Optimisation: éviter l'allocation de string
			ctx.Response.Header.SetServerBytes([]byte("Velocity"))
			routerHandler(ctx)
		}
	}

	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.SetServerBytes([]byte("Velocity"))

		// Utiliser le pool de Context
		c := contextPool.Get().(*types.Context)
		c.RequestCtx = ctx
		defer func() {
			c.RequestCtx = nil // éviter les fuites mémoire
			contextPool.Put(c)
		}()

		handler := a.chain(func(c *types.Context) error {
			// Cette partie sera gérée par le router directement
			routerHandler(ctx)
			return nil
		})

		handler(c)
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
		ReduceMemoryUsage:             false, // false pour de meilleures performances
		TCPKeepalive:                  true,
		NoDefaultServerHeader: true, 
		NoDefaultDate:         true,
		NoDefaultContentType:  true, 
	}

	log.Printf("Server starting on %s with %d max goroutines", addr, a.config.Concurrency)
	return server.ListenAndServe(addr)
}
