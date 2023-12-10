package giris2

import (
	"net/http"

	"github.com/kataras/iris/v12"

	j2rpc "github.com/atcharles/glibs/j2rpc2"
)

func HttpHandler2IrisHandler(h http.Handler) iris.Handler {
	return func(c iris.Context) {
		h.ServeHTTP(c.ResponseWriter(), c.Request())
	}
}

func RPCServer2IrisHandler(s j2rpc.Server) iris.Handler {
	return func(c iris.Context) {
		s.Handler(j2rpc.NewContext(c, c.ResponseWriter(), c.Request()))
	}
}
