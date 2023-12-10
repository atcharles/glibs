package j2rpc

import (
	"reflect"
	"time"
)

type callback struct {
	server *server
	//请求方提供的方法
	providerMethod string
	//执行的方法名
	methodName string
	//receiver object value
	rcv reflect.Value
	//the function
	fn reflect.Value
	//input argument types
	argTypes []reflect.Type
	//method's first argument is a context (not included in argTypes)
	hasCtx bool
	//err return idx, of -1 when method cannot return error
	errPos int
}

// call invokes the callback.
func (c *callback) call(ctx *Context, args []reflect.Value) (res interface{}, err error) {
	//Create the argument slice.
	fullArgs := make([]reflect.Value, 0, 2+len(args))
	if c.rcv.IsValid() {
		fullArgs = append(fullArgs, c.rcv)
	}
	if c.hasCtx {
		fullArgs = append(fullArgs, reflect.ValueOf(ctx.Context))
	}
	fullArgs = append(fullArgs, args...)

	//Catch panic while running the callback.
	defer func() {
		if p := recover(); p != nil {
			err = c.server.stack(p, c.providerMethod)
			return
		}
	}()

	if c.server.callBefore != nil {
		if err = c.server.callBefore(ctx, c.providerMethod, args); err != nil {
			return
		}
	}

	// call before
	if v, ok := c.rcv.Interface().(ItfBeforeCall); ok {
		if err = v.BeforeCall(ctx, c.providerMethod, args); err != nil {
			return
		}
	}

	//Run the callback.
	results := c.fn.Call(fullArgs)

	ctx.Store.Store(EndTimeKey, time.Now().UnixNano())

	if _val, ok := ctx.Store.Load(StartTimeKey); ok {
		_startTime := _val.(int64)
		ctx.Store.Store(TimeUsedKey, time.Now().UnixNano()-_startTime)
	}

	// call after
	if v, ok := c.rcv.Interface().(ItfAfterCall); ok {
		if err = v.AfterCall(ctx, c.providerMethod, args, results); err != nil {
			return
		}
	}

	if c.server.callAfter != nil {
		if err = c.server.callAfter(ctx, c.providerMethod, args, results); err != nil {
			return
		}
	}

	if len(results) == 0 {
		return
	}
	if c.errPos >= 0 {
		err = value2err(results[c.errPos])
		if err != nil {
			return
		}
	}
	rv := results[0]
	if !rv.IsValid() {
		return
	}
	return rv.Interface(), err
}

// makeArgTypes ...
func (c *callback) makeArgTypes() bool {
	fnt := c.fn.Type()

	outs := make([]reflect.Type, fnt.NumOut())
	for i := 0; i < fnt.NumOut(); i++ {
		outs[i] = fnt.Out(i)
	}
	//A maximum of two values can be returned.
	if len(outs) > 2 {
		return false
	}
	//If an error is returned, it must be the last returned value.
	switch {
	case len(outs) == 1 && isErrorType(outs[0]):
		c.errPos = 0
	case len(outs) == 2:
		if isErrorType(outs[0]) || !isErrorType(outs[1]) {
			return false
		}
		c.errPos = 1
	}

	firstArg := 0
	if c.rcv.IsValid() {
		firstArg++
	}

	if fnt.NumIn() > firstArg && fnt.In(firstArg).Implements(contextType) {
		c.hasCtx = true
		firstArg++
	}
	//Add all remaining parameters.
	c.argTypes = make([]reflect.Type, fnt.NumIn()-firstArg)
	for i := firstArg; i < fnt.NumIn(); i++ {
		c.argTypes[i-firstArg] = fnt.In(i)
	}
	return true
}

func value2err(val reflect.Value) error {
	if !isErrorType(val.Type()) {
		return nil
	}
	if !val.IsValid() {
		return nil
	}
	if val.IsZero() {
		return nil
	}
	return val.Interface().(error)
}
