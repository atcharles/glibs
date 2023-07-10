package j2rpc

import (
	"log"
	"net/http"
	"reflect"
)

type (
	//ItfConstructor ...
	ItfConstructor interface {
		Constructor()
	}
	//ItfExcludeMethod ...
	ItfExcludeMethod interface {
		ExcludeMethod() []string
	}
	//RPCServer ...
	RPCServer interface {
		SetCallBefore(before BeforeHandler)
		SetCallAfter(after AfterHandler)
		GetContext() *Context
		Use(handlers ...Handler)
		Opt() *Option
		Logger() LevelLogger
		SetLogger(logger LevelLogger)
		ServeHTTP(w http.ResponseWriter, r *http.Request)
		RegisterForApp(app interface{})
		Register(receiver interface{}, names ...string)
		Handler(ctx *Context)
		Stop()
		NamespaceService(method string) interface{}
	}
	//ItfNamespaceName ...
	ItfNamespaceName interface {
		J2rpcNamespaceName() string
	}

	ItfBeforeCall interface {
		BeforeCall(ctx *Context, method string, args []reflect.Value) error
	}

	ItfAfterCall interface {
		AfterCall(ctx *Context, method string, args []reflect.Value, results []reflect.Value) error
	}

	ItfStartup interface {
		Startup(rs RPCServer)
	}
)

// PopulateConstructor ...
func PopulateConstructor(value interface{}, args ...interface{}) {
	debug := len(args) > 0 && args[0] == true
	vp := reflect.ValueOf(value)
	if vp.Kind() != reflect.Ptr {
		panic("need pointer")
	}

	if v, ok := value.(ItfConstructor); ok {
		v.Constructor()
		if debug {
			log.Printf("%s.Constructor executed\n", vp.Type().String())
		}
	}

	for vp.Kind() == reflect.Ptr {
		vp = vp.Elem()
	}

	for i := 0; i < vp.NumField(); i++ {
		fv1 := vp.Field(i)
		if fv1.Kind() != reflect.Ptr {
			continue
		}
		if fv1v, ok := fv1.Interface().(ItfConstructor); ok {
			fv1v.Constructor()
			if debug {
				log.Printf("%s.Constructor executed\n", fv1.Type().String())
			}
		}
	}
}
