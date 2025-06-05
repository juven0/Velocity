package core

import (
	"sync"

	"github.com/juven0/Velocity/types"
	"github.com/valyala/fasthttp"
)

var contextPool = sync.Pool{
	New: func() interface{} {
		return &types.Context{}
	},
}

func GetContext(ctx *fasthttp.RequestCtx) *types.Context {
	c := contextPool.Get().(*types.Context)
	c.RequestCtx = ctx
	return c
}

func ReleasContext(c *types.Context) {
	c.RequestCtx = nil
	contextPool.Put(c)
}
