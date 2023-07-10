package giris

import (
	"net/http"

	"github.com/kataras/iris/v12"

	"github.com/atcharles/glibs/j2rpc"
)

func HttpHandler2IrisHandler(h http.Handler) iris.Handler {
	return func(c iris.Context) {
		h.ServeHTTP(c.ResponseWriter(), c.Request())
	}
}

func RPCServer2IrisHandler(h j2rpc.RPCServer) iris.Handler {
	return func(c iris.Context) {
		h.Handler(h.GetContext().BeginWithContext(c, c.ResponseWriter(), c.Request()))
	}
}
