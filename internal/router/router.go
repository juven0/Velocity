package router

import (
	"hash/fnv"
	"strings"
	"sync"
	"unsafe"

	"github.com/juven0/Velocity/types"
	"github.com/valyala/fasthttp"
)

type FixedParams struct {
	keys   [8]string
	values [8]string
	count  int
}

func (fp *FixedParams) Set(key, value string) {
	if fp.count < 8 {
		fp.keys[fp.count] = key
		fp.values[fp.count] = value
		fp.count++
	}
}

func (fp *FixedParams) Get(key string) (string, bool) {
	for i := 0; i < fp.count; i++ {
		if fp.keys[i] == key {
			return fp.values[i], true
		}
	}
	return "", false
}

func (fp *FixedParams) Reset() {
	fp.count = 0
}

func (fp *FixedParams) ToMap() map[string]string {
	if fp.count == 0 {
		return nil
	}
	m := make(map[string]string, fp.count)
	for i := 0; i < fp.count; i++ {
		m[fp.keys[i]] = fp.values[i]
	}
	return m
}

type lockFreeCache struct {
	entries [1024]*routeCacheEntry
	mask    uint32
}

type routeCacheEntry struct {
	key    string
	node   *node
	params *FixedParams
}

func newLockFreeCache() *lockFreeCache {
	return &lockFreeCache{
		mask: 1023, // 1024 - 1
	}
}

func (lfc *lockFreeCache) Get(key string) (*routeCacheEntry, bool) {
	hash := hashString(key)
	idx := hash & lfc.mask
	entry := lfc.entries[idx]
	if entry != nil && entry.key == key {
		return entry, true
	}
	return nil, false
}

func (lfc *lockFreeCache) Set(key string, node *node, params *FixedParams) {
	hash := hashString(key)
	idx := hash & lfc.mask

	entry := &routeCacheEntry{
		key:  key,
		node: node,
	}

	if params != nil && params.count > 0 {
		entry.params = &FixedParams{}
		*entry.params = *params
	}

	lfc.entries[idx] = entry
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

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
	trees     map[string]*node
	cache     *lockFreeCache
	cacheSize int
}

type Groupe struct {
	prefix string
	router *Router
}

var (
	fixedParamsPool = sync.Pool{
		New: func() interface{} {
			return &FixedParams{}
		},
	}

	cacheKeyPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 64)
		},
	}

	pathPool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 8)
		},
	}

	contextPool = sync.Pool{
		New: func() interface{} {
			return &types.Context{}
		},
	}
)

func getCacheKey(method, path string) string {
	buf := cacheKeyPool.Get().([]byte)
	buf = buf[:0]
	buf = append(buf, method...)
	buf = append(buf, ':')
	buf = append(buf, path...)
	key := string(buf)
	cacheKeyPool.Put(buf)
	return key
}

func New() *Router {
	return &Router{
		trees:     make(map[string]*node, 16),
		cache:     newLockFreeCache(),
		cacheSize: 1024,
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
		c := contextPool.Get().(*types.Context)
		c.RequestCtx = ctx
		defer func() {
			c.RequestCtx = nil
			contextPool.Put(c)
		}()

		method := ctx.Method()
		path := ctx.Path()

		methodStr := bytesToString(method)
		pathStr := bytesToString(path)

		cacheKey := getCacheKey(methodStr, pathStr)

		if cached, exists := r.cache.Get(cacheKey); exists {
			if cached.node != nil && cached.node.handler != nil {
				if cached.params != nil && cached.params.count > 0 {
					ctx.SetUserValue("params", cached.params.ToMap())
				}
				cached.node.handler(c)
				return
			}
		}

		n := r.trees[methodStr]
		if n == nil {
			ctx.Error("Not Found", fasthttp.StatusNotFound)
			return
		}

		if len(path) <= 1 {
			if n.handler != nil {
				n.handler(c)
				r.cache.Set(cacheKey, n, nil)
				return
			}
			ctx.Error("Not Found", fasthttp.StatusNotFound)
			return
		}

		params := fixedParamsPool.Get().(*FixedParams)
		params.Reset()
		defer fixedParamsPool.Put(params)

		matched := r.matchPathOptimized(n, path[1:], params)
		if matched == nil || matched.handler == nil {
			ctx.Error("Not Found", fasthttp.StatusNotFound)
			return
		}

		if params.count > 0 {
			ctx.SetUserValue("params", params.ToMap())
		}

		if params.count == 0 {
			r.cache.Set(cacheKey, matched, nil)
		}

		matched.handler(c)
	}
}

func (r *Router) matchPathOptimized(n *node, path []byte, params *FixedParams) *node {
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
		params.Set(paramName, segmentStr)
		return r.matchPathOptimized(n.paramChild, remaining, params)
	}

	if n.wildcardChild != nil {
		params.Set("*", bytesToString(path))
		return n.wildcardChild
	}

	return nil
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
