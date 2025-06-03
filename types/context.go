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

func (c *Context) SetStatus(code int) {
	c.SetStatusCode(code)
}

func (c *Context) Query(key string) string {
	return string(c.QueryArgs().Peek(key))
}

// func (c *Context) Param(key string)
func (c *Context) FormValue(key string) string {
	return ""
}

func (c *Context) SetHeader(key, value string) {
	c.Response.Header.Set(key, value)
}

func (c *Context) Redirect(url string, code int) {
	c.Response.Header.Set("Location", url)
	c.SetStatusCode(code)
}

func (c *Context) Body() []byte {
	return c.Request.Body()
}

func (c *Context) BodyString() string {
	return string(c.Request.Body())
}

func (c *Context) BindJSON(v interface{}) error {
	return json.Unmarshal(c.Body(), v)
}
