package j2rpc

import (
	"sync"
)

func newContextPool(newFunc func() interface{}) *ContextPool {
	return &ContextPool{Pool: &sync.Pool{New: newFunc}}
}

type ContextPool struct {
	*sync.Pool
}

// Acquire ...
func (p *ContextPool) Acquire() *Context {
	return p.Pool.Get().(*Context)
}

// Release ...
func (p *ContextPool) Release(c *Context) {
	c.msg.empty()
	p.Pool.Put(c)
}
