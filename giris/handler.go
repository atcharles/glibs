package giris

import (
	"errors"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/spf13/cast"
)

var (
	contextType = reflect.TypeOf((iris.Context)(nil))
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

// JSONResponse ...
type JSONResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func BindRestAPIs(g iris.Party, api interface{}) {
	supportMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE"}
	fn1add2router := func(party iris.Party, method reflect.Method) {
		for _, v := range supportMethods {
			if strings.HasPrefix(strings.ToUpper(method.Name), v) {
				routeStr := "/" + method.Name[len(v):]
				party.Handle(v, routeStr, Func2Handler(method.Func.Interface()))
				return
			}
		}
		party.Any("/"+method.Name, Func2Handler(method.Func.Interface()))
	}
	api1t := reflect.TypeOf(api)
	// scan all methods
	for i := 0; i < api1t.NumMethod(); i++ {
		fn1add2router(g, api1t.Method(i))
	}
}

func Func2Handler(fn interface{}) iris.Handler {
	return func(c iris.Context) {
		val, err := parseArgumentFromContext(fn, c)
		if err != nil {
			JSON(c, 400, err.Error())
			return
		}
		JSON(c, 200, val)
	}
}

// JSON ...
func JSON(c iris.Context, code int, val ...interface{}) {
	var msg string
	var data interface{}
	if len(val) > 0 {
		_fn1 := func() {
			if code != 200 {
				msg = cast.ToString(val[0])
				return
			}
			data = val[0]
		}
		_fn1()
	}
	_ = c.JSON(&JSONResponse{Code: code, Msg: msg, Data: data})
}

func parseArgumentFromContext(fn interface{}, c iris.Context) (val interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(cast.ToString(e))
		}
	}()
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return nil, errors.New("need func type")
	}
	args := make([]reflect.Value, 0)

	numIn := fnv.Type().NumIn()
	if numIn == 3 {
		// 3个参数的情况，第一个参数是 receiver
		args = append(args, reflectNewByType(fnv.Type().In(0)))
	}

	if numIn > 3 {
		return nil, errors.New("too many arguments, want at most 2")
	}

	for i := len(args); i < fnv.Type().NumIn(); i++ {
		arg1 := fnv.Type().In(i)
		if arg1.AssignableTo(contextType) {
			args = append(args, reflect.ValueOf(c))
			continue
		}
		args = append(args, reflectNewByType(arg1, func(v interface{}) {
			err = c.ReadBody(v)
		}))
		if err != nil {
			return
		}
	}

	if len(args) != numIn {
		return nil, errors.New("arguments not match")
	}
	results := fnv.Call(args)
	if len(results) == 0 {
		val = "OK"
		return
	}
	// if the end of result is error type
	end := results[len(results)-1]
	if end.Type().Implements(errorType) {
		results = results[:len(results)-1]
		if err = value2err(end); err != nil {
			return
		}
		if len(results) == 0 {
			val = "OK"
			return
		}
	}
	if len(results) == 1 {
		val = results[0].Interface()
		return
	}
	data := make([]interface{}, 0, len(results))
	for _, v := range results {
		data = append(data, v.Interface())
	}
	val = data
	return
}

func reflectNewByType(t reflect.Type, binds ...func(v interface{})) reflect.Value {
	var bind func(v interface{})
	if len(binds) > 0 {
		bind = binds[0]
	}
	var v reflect.Value
	switch t.Kind() {
	case reflect.Ptr:
		v = reflect.New(t.Elem())
		if bind != nil {
			bind(v.Interface())
		}
	default:
		v = reflect.New(t)
		if bind != nil {
			bind(v.Interface())
		}
		v = v.Elem()
	}
	return v
}

func value2err(val reflect.Value) error {
	if !val.Type().Implements(errorType) {
		return nil
	}
	if !val.IsValid() {
		return nil
	}
	if val.IsZero() {
		return nil
	}
	vv, ok := val.Interface().(error)
	if !ok {
		return nil
	}
	return vv
}
