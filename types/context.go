package types

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

type Context struct {
	*fasthttp.RequestCtx
	next HandlerFunc
}

func (c *Context) Text(msg string) error {
	c.SetContentType("text/plain")
	c.SetBodyString(msg)
	return nil
}

func (c *Context) JSON(v interface{}) error {
	c.SetContentType("application/json")
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.SetBody(b)
	return nil
}

func (c *Context) WithNext(next HandlerFunc) *Context {
	return &Context{RequestCtx: c.RequestCtx, next: next}
}

func (c *Context) Next() error {
	if c.next != nil {
		return c.next(c)
	}

	return nil
}
