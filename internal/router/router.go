package router

import (
	"strings"
	"sync"
	"unsafe"

	"github.com/juven0/Velocity/types"
	"github.com/valyala/fasthttp"
)

type node struct {
	part           string
	children       []*node
	handler        HandlerFunc
	param          bool
	wildcard       bool
	staticChildren map[string]*node
	paramChild     *node
	wildcardChild  *node
	isLeaf         bool
}

type HandlerFunc = types.HandlerFunc

type Router struct {
	trees      map[string]*node
	routeCache map[string]*routeCacheEntry
	cacheMu    sync.RWMutex
	cacheSize  int
}

type routeCacheEntry struct {
	node   *node
	params map[string]string
}

type Groupe struct {
	prefix string
	router *Router
}

var (
	paramsPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]string, 4)
		},
	}

	pathPool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 8)
		},
	}

	cacheEntryPool = sync.Pool{
		New: func() interface{} {
			return &routeCacheEntry{
				params: make(map[string]string, 4),
			}
		},
	}
)

func New() *Router {
	return &Router{
		trees:      make(map[string]*node, 16),
		routeCache: make(map[string]*routeCacheEntry, 1024),
		cacheSize:  1024,
	}
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
		r.trees[method] = &node{
			staticChildren: make(map[string]*node, 8),
		}
	}

	parts := r.fastSplitPath(path[1:])
	current := r.trees[method]

	for _, part := range parts {
		child := r.findOrCreateChild(current, part)
		current = child
	}
	current.handler = handler
	current.isLeaf = true

	parts = parts[:0]
	pathPool.Put(parts)
}

func (r *Router) fastSplitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	parts := pathPool.Get().([]string)
	parts = parts[:0]

	pathBytes := *(*[]byte)(unsafe.Pointer(&path))

	start := 0
	for i := 0; i <= len(pathBytes); i++ {
		if i == len(pathBytes) || pathBytes[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}

	return parts
}

func (r *Router) findOrCreateChild(parent *node, part string) *node {
	if parent.staticChildren == nil {
		parent.staticChildren = make(map[string]*node, 4)
	}

	if child, exists := parent.staticChildren[part]; exists {
		return child
	}

	if strings.HasPrefix(part, ":") {
		if parent.paramChild != nil {
			return parent.paramChild
		}
		child := &node{
			part:           part,
			param:          true,
			staticChildren: make(map[string]*node, 4),
		}
		parent.paramChild = child
		parent.children = append(parent.children, child)
		return child
	}

	if part == "*" {
		if parent.wildcardChild != nil {
			return parent.wildcardChild
		}
		child := &node{
			part:           part,
			wildcard:       true,
			staticChildren: make(map[string]*node, 4),
		}
		parent.wildcardChild = child
		parent.children = append(parent.children, child)
		return child
	}

	child := &node{
		part:           part,
		staticChildren: make(map[string]*node, 4),
	}
	parent.staticChildren[part] = child
	parent.children = append(parent.children, child)
	return child
}

func (r *Router) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		method := ctx.Method()
		path := ctx.Path()

		methodStr := bytesToString(method)
		pathStr := bytesToString(path)

		cacheKey := methodStr + ":" + pathStr

		r.cacheMu.RLock()
		if cached, exists := r.routeCache[cacheKey]; exists {
			r.cacheMu.RUnlock()

			if cached.node != nil && cached.node.handler != nil {
				c := &types.Context{RequestCtx: ctx}
				if len(cached.params) > 0 {
					ctx.SetUserValue("params", cached.params)
				}
				cached.node.handler(c)
				return
			}
		}
		r.cacheMu.RUnlock()

		n := r.trees[methodStr]
		if n == nil {
			ctx.Error("Not Found", fasthttp.StatusNotFound)
			return
		}

		if len(path) <= 1 {
			if n.handler != nil {
				c := &types.Context{RequestCtx: ctx}
				n.handler(c)
				// Mettre en cache
				r.cacheRoute(cacheKey, n, nil)
				return
			}
			ctx.Error("Not Found", fasthttp.StatusNotFound)
			return
		}

		params := paramsPool.Get().(map[string]string)
		defer r.releaseParams(params)

		matched := r.matchPathOptimized(n, path[1:], params)
		if matched == nil || matched.handler == nil {
			ctx.Error("Not Found", fasthttp.StatusNotFound)
			return
		}

		c := &types.Context{RequestCtx: ctx}
		if len(params) > 0 {
			ctx.SetUserValue("params", params)
		}

		if len(params) == 0 {
			r.cacheRoute(cacheKey, matched, nil)
		}

		matched.handler(c)
	}
}

func (r *Router) matchPathOptimized(n *node, path []byte, params map[string]string) *node {
	if len(path) == 0 {
		return n
	}

	segmentEnd := 0
	for segmentEnd < len(path) && path[segmentEnd] != '/' {
		segmentEnd++
	}

	segment := path[:segmentEnd]
	remaining := path[segmentEnd:]
	if len(remaining) > 0 {
		remaining = remaining[1:]
	}

	segmentStr := bytesToString(segment)

	if child, exists := n.staticChildren[segmentStr]; exists {
		return r.matchPathOptimized(child, remaining, params)
	}

	if n.paramChild != nil {
		paramName := n.paramChild.part[1:]
		params[paramName] = segmentStr
		return r.matchPathOptimized(n.paramChild, remaining, params)
	}

	if n.wildcardChild != nil {
		params["*"] = bytesToString(path)
		return n.wildcardChild
	}

	return nil
}

func (r *Router) cacheRoute(key string, node *node, params map[string]string) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	if len(r.routeCache) >= r.cacheSize {
		// Simple éviction: supprimer une entrée aléatoire
		for k := range r.routeCache {
			delete(r.routeCache, k)
			break
		}
	}

	entry := cacheEntryPool.Get().(*routeCacheEntry)
	entry.node = node

	if len(params) > 0 {
		for k, v := range params {
			entry.params[k] = v
		}
	}

	r.routeCache[key] = entry
}

func (r *Router) releaseParams(params map[string]string) {
	for k := range params {
		delete(params, k)
	}
	paramsPool.Put(params)
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func (r *Router) GET(path string, handler HandlerFunc) {
	r.Handel("GET", path, handler)
}

func (r *Router) POST(path string, handler HandlerFunc) {
	r.Handel("POST", path, handler)
}

func (r *Router) PUT(path string, handler HandlerFunc) {
	r.Handel("PUT", path, handler)
}

func (r *Router) DELETE(path string, handler HandlerFunc) {
	r.Handel("DELETE", path, handler)
}

func (r *Router) PATCH(path string, handler HandlerFunc) {
	r.Handel("PATCH", path, handler)
}

func (r *Router) OPTIONS(path string, handler HandlerFunc) {
	r.Handel("OPTIONS", path, handler)
}

func (r *Router) HEAD(path string, handler HandlerFunc) {
	r.Handel("HEAD", path, handler)
}

// Méthodes pour les groupes
func (g *Groupe) GET(path string, handler HandlerFunc) {
	g.Handel("GET", path, handler)
}

func (g *Groupe) POST(path string, handler HandlerFunc) {
	g.Handel("POST", path, handler)
}

func (g *Groupe) PUT(path string, handler HandlerFunc) {
	g.Handel("PUT", path, handler)
}

func (g *Groupe) DELETE(path string, handler HandlerFunc) {
	g.Handel("DELETE", path, handler)
}

func (g *Groupe) PATCH(path string, handler HandlerFunc) {
	g.Handel("PATCH", path, handler)
}

func (g *Groupe) OPTIONS(path string, handler HandlerFunc) {
	g.Handel("OPTIONS", path, handler)
}

func (g *Groupe) HEAD(path string, handler HandlerFunc) {
	g.Handel("HEAD", path, handler)
}
