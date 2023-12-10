package giris

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/middleware/rate"

	"github.com/atcharles/glibs/j2rpc"
)

func ConcurrentLimit(n int) iris.Handler {
	ch := make(chan struct{}, n)
	return func(c iris.Context) {
		select {
		case ch <- struct{}{}:
			c.Next()
			<-ch
		default:
			_ = c.StopWithJSON(http.StatusTooManyRequests, &j2rpc.RPCMessage{
				ID:    json.RawMessage("1"),
				Error: j2rpc.NewError(http.StatusTooManyRequests, "系统繁忙,请稍后再试"),
			})
		}
	}
}

func RateLimiter() context.Handler {
	return rate.Limit(
		10,
		30,
		rate.PurgeEvery(5*time.Minute, 15*time.Minute),
		rate.ExceedHandler(func(c iris.Context) {
			_ = c.StopWithJSON(http.StatusTooManyRequests, &j2rpc.RPCMessage{
				ID:    json.RawMessage("1"),
				Error: j2rpc.NewError(http.StatusTooManyRequests, "请求过于频繁"),
			})
		}),
	)
}
