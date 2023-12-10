package j2rpc

import "reflect"

type AfterHandler func(ctx *Context, method string, params []reflect.Value, result []reflect.Value) (err error)

type BeforeHandler func(ctx *Context, method string, params []reflect.Value) (err error)

type Handler func(c *Context) (err error)

type Handlers []Handler

func UpsertHandlers(handlers Handlers, newHandlers Handlers) Handlers {
	for _, h := range newHandlers {
		fn1 := FuncNameFull(h)
		var duplicate = false
		for _, nh := range handlers {
			fn2 := FuncNameFull(nh)
			if fn1 == fn2 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			handlers = append(handlers, h)
		}
	}
	return handlers
}
