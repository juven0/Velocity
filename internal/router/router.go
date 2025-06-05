package router

import (
	"strings"
	"unsafe"

	"github.com/juven0/Velocity/internal/core"
	"github.com/juven0/Velocity/types"
	"github.com/valyala/fasthttp"
)

type node struct {
	part     string
	children []*node
	handler  HandlerFunc
	param    bool
	wildcard bool
}

type HandlerFunc = types.HandlerFunc

type Router struct {
	trees map[string]*node
}

type Groupe struct {
	prefix string
	router *Router
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func New() *Router {
	return &Router{trees: map[string]*node{}}
}

func (r *Router) Groupe(prefix string) *Groupe {
	return &Groupe{
		prefix: prefix,
		router: r,
	}
}

func (g *Groupe) Handel(method string, path string, handler HandlerFunc) {
	fullPath := g.prefix + path
	g.router.Handel(method, fullPath, handler)
}

func (r *Router) Handel(method string, path string, handler HandlerFunc) {
	if path == "" || path[0] != '/' {
		panic("path must start with '/'")
	}

	if r.trees[method] == nil {
		r.trees[method] = &node{}
	}

	parts := strings.Split(path[1:], "/")
	current := r.trees[method]

	for _, part := range parts {
		var child *node
		for _, c := range current.children {
			if c.part == part || c.param || c.wildcard {
				child = c
				break
			}
		}
		if child == nil {
			child = &node{part: part}
			if strings.HasPrefix(part, ":") {
				child.param = true
			} else if part == "*" {
				child.wildcard = true
			}
			current.children = append(current.children, child)
		}
		current = child
	}
	current.handler = handler
}

func (r *Router) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		method := string(ctx.Method())
		pathBytes := ctx.Path()
		path := bytesToString(pathBytes)

		parts := strings.Split(strings.Trim(path, "/"), "/")
		n := r.trees[method]
		params := make(map[string]string)

		for _, part := range parts {
			var matched *node
			for _, c := range n.children {
				if c.part == part || c.param || c.wildcard {
					matched = c
					if c.param {
						params[c.part[1:]] = part
					} else if c.wildcard {
						params["*"] = strings.Join(parts, "/")
					}
					break
				}
			}
			if matched == nil {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
				return
			}
			n = matched
		}

		if n.handler != nil {
			c := core.GetContext(ctx)
			defer core.ReleasContext(c)
			ctx.SetUserValue("params", params)
			n.handler(c)
		} else {
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}
	}
}
