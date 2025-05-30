package velocity

import (
	"github.com/valyala/fasthttp"
)

type App struct{}

func New() *App {
	return &App{}
}

func (a *App) Get(path string, handler Hanbler) {
}

func (a *App) Listen(addr string) error {
	return fasthttp.ListenAndServe()
}
