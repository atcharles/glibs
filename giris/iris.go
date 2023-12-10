package giris

import (
	stdContext "context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/host"
	"github.com/kataras/iris/v12/middleware/cors"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/rate"
	recover2 "github.com/kataras/iris/v12/middleware/recover"

	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/j2rpc"
	"github.com/atcharles/glibs/util"
)

var App = new(AppStart)

type AppStart struct {
	IrisApp   *iris.Application `inject:""`
	Config    *config.Config    `inject:""`
	WebLogger *midLogger        `inject:""`
	Captcha   *Captcha          `inject:""`
	JWT       *JWT              `inject:""`
	Validator *util.Validator   `inject:""`
	RPC       j2rpc.RPCServer   `inject:""`
	SysLogger util.ItfLogger    `inject:""`

	rootParty iris.Party

	stop chan struct{}
	//done chan struct{}

	RPCServerFunc func(server j2rpc.RPCServer)
}

func (a *AppStart) Constructor() {
	a.stop = make(chan struct{}, 1)
	//a.done = make(chan struct{}, 1)

	a.WebLogger.Constructor()
	a.IrisApp = iris.New()
	a.defaultMiddlewares(a.IrisApp)
	a.RouteApp()
}

func (a *AppStart) RootParty() iris.Party {
	return a.rootParty
}

// RouteApp ......
func (a *AppStart) RouteApp() iris.Party {
	p := a.IrisApp.Party("/")
	p.Use(
		logger.New(),
		func(c iris.Context) {
			c.Header("X-Server", "")
			c.Header("X-Server", "lucky")
			c.Next()
		},
		ConcurrentLimit(a.Config.V().GetInt("server.concurrent_limit")),
		func(c iris.Context) {
			if jt := c.Values().GetString(JWTToken); jt != "" {
				rate.SetIdentifier(c, fmt.Sprintf("%s:%s", c.Path(), jt))
				c.Next()
				return
			}
			rate.SetIdentifier(c, fmt.Sprintf("%s:%s", c.Path(), c.RemoteAddr()))
			c.Next()
		},
		RateLimiter(),
	)
	p.Get("/ping", func(c iris.Context) { _, _ = c.WriteString("pong") })
	p.Get("/captcha", a.Captcha.Captcha())
	if a.RPCServerFunc != nil {
		a.RPCServerFunc(a.RPC)
	}
	a.RPC.SetLogger(a.WebLogger.WebLogger())
	if str := strings.TrimSpace(config.Viper().GetString("server.crypto")); str != "" {
		a.RPC.Opt().CryptoKey = str
	}
	p.Post("/rpc", iris.Compression, RPCServer2IrisHandler(a.RPC))
	a.rootParty = p
	return p
}

// Shutdown ...
func (a *AppStart) Shutdown() {
	close(a.stop)
}

// StartApp ......
func (a *AppStart) StartApp() {
	//go get -u github.com/google/pprof
	//需要安装 graphviz
	//pprof -http=:8080 http://127.0.0.1:8080/debug/pprof/profile\?seconds\=10
	pprofServer := &http.Server{
		Addr: a.Config.V().GetString("server.pprof_port"),
	}
	go func() { _ = pprofServer.ListenAndServe() }()

	serverHost := a.Config.V().GetString("server.host")
	go func() {
		a.IrisApp.ConfigureHost(func(su *host.Supervisor) {
			su.RegisterOnShutdown(func() {
				//a.WebLogger.WebLogger().Println("[iris app] Shutdown...")
				log.Println("[iris app] Shutdown...")
				ctx := stdContext.Background()
				_ = pprofServer.Shutdown(ctx)
				<-a.stop
			})
		})

		err := a.IrisApp.Run(
			Addr(serverHost),
			iris.WithoutServerError(iris.ErrServerClosed),
			iris.WithSocketSharding,
			iris.WithKeepAlive(time.Second*60),
			iris.WithTimeout(time.Second*10),
			iris.WithOptimizations,
		)
		if err != nil {
			log.Printf("ServerStart start error: %s\n", err.Error())
			return
		}
	}()
}

// Wait ...
func (a *AppStart) Wait() {
	<-a.stop
}

func (a *AppStart) defaultMiddlewares(p iris.Party) {
	p.Logger().SetTimeFormat("2006-01-02 15:04:05.000")
	p.Logger().SetOutput(a.WebLogger.WebLogger().Writer())
	p.Logger().SetLevel(a.Config.V().GetString("app.log_level"))
	p.UseRouter(
		recover2.New(),
		cors.New().ExposeHeaders(
			"X-Server",
			"Authorization",
			"X-Authorization",
			"Request-Id",
			"X-Request-Id",
		).AllowHeaders(
			"Authorization",
			"X-Authorization",
			"Request-Id",
			"X-Request-Id",
			"X-Server",
			"token",
			"Accept",
			"Accept-Language",
			"Content-Language",
			"Content-Type",
		).Handler(),
	)
}

func Addr(addr string, hostConfigs ...host.Configurator) iris.Runner {
	return func(app *iris.Application) error {
		return app.NewHost(AddrServer(addr)).
			Configure(hostConfigs...).
			ListenAndServe()
	}
}

func AddrServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		ReadTimeout:       time.Second * 10,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 10,
		IdleTimeout:       time.Second * 60,
	}
}
