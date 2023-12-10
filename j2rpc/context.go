package j2rpc

import (
	"context"
	"net/http"
	"sync"
)

const (
	StartTimeKey = "__j2rpc_startTimeKey"
	EndTimeKey   = "__j2rpc_endTimeKey"
	TimeUsedKey  = "__j2rpc_timeUsedKey"
)

type Context struct {
	context.Context
	app          *server
	w            http.ResponseWriter
	r            *http.Request
	isStopped    bool
	currentIndex int
	msg          *RPCMessage
	midHandlers  Handlers

	Body  []byte
	Store sync.Map
}

func (c *Context) App() RPCServer { return c.app }

// BeginRequest ......
func (c *Context) BeginRequest(w http.ResponseWriter, r *http.Request) *Context {
	c.w = w
	c.r = r
	c.isStopped = false
	c.currentIndex = 0
	return c
}

// BeginWithContext ... start a new context with the given context
func (c *Context) BeginWithContext(ctx context.Context, w http.ResponseWriter, r *http.Request) *Context {
	return c.SetContext(ctx).BeginRequest(w, r)
}

// Do ......
func (c *Context) Do(handlers ...Handler) error {
	if len(handlers) == 0 {
		return nil
	}
	c.midHandlers = handlers
	return c.midHandlers[0](c)
}

// IsStopped ......
func (c *Context) IsStopped() bool { return c.isStopped }

// Method ......
func (c *Context) Method() string { return c.msg.Method }

func (c *Context) Msg() *RPCMessage { return c.msg }

func (c *Context) Next() (err error) {
	if c.IsStopped() {
		return
	}
	c.currentIndex++
	// 1. check if the current index is the last one
	if c.currentIndex >= len(c.midHandlers) {
		return
	}
	return c.midHandlers[c.currentIndex](c)
}

// Release ......
func (c *Context) Release() {
	c.app.contextPool.Release(c)
}

// Request ......
func (c *Context) Request() *http.Request { return c.r }

// SetContext ......
func (c *Context) SetContext(ctx context.Context) *Context {
	c.Context = ctx
	return c
}

func (c *Context) SetMsg(msg *RPCMessage) { c.msg = msg }

// Stop ......
func (c *Context) Stop() { c.isStopped = true }

// Writer ......
func (c *Context) Writer() http.ResponseWriter { return c.w }

// newContext ......
func newContext(app *server) *Context {
	return &Context{
		Context: context.Background(),
		app:     app,
		msg:     &RPCMessage{},
	}
}
