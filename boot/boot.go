package boot

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/kataras/iris/v12"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"

	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/gemq"
	"github.com/atcharles/glibs/giris"
	"github.com/atcharles/glibs/j2rpc"
	"github.com/atcharles/glibs/mdb"
	"github.com/atcharles/glibs/util"
)

var ServiceApp interface{}
var (
	_ = RegisterBeforeInject
	_ = RegisterAfterInject
	_ = RegisterBeforeHttpServer
	_ = RegisterAfterHttpServer
	_ = RegisterShutdown
)
var (
	_ = spew.Dump
	_ = assert.Equal
)
var (
	beforeInjectMap     = make(map[string]func())
	afterInjectMap      = make(map[string]func())
	beforeHttpServerMap = make(map[string]func())
	afterHttpServerMap  = make(map[string]func())
	shutdownMap         = make(map[string]func())
)
var (
	mapType = util.AnyType[map[string]interface{}]()
)

type Callback func()

type ConfigFunc func(c *config.Config)

func Drop() {
	redisOpt := parseRedisOpt()
	if redisOpt != nil {
		mdb.Rdb.Initialize(redisOpt)
		mdb.Rdb.GetClient().FlushAll(context.Background())
	}
	dbOpt := parseDBOpt()
	if dbOpt != nil {
		mdb.DB.SetOpt(dbOpt)
		mdb.DB.DropDB()
	}
	config.C.SetValue("app.runtimes", 1)
}

// LoadConfig load config with args ... [ string, ConfigFunc, Callback, any type convertible to map[string]interface{} ]
func LoadConfig(args ...interface{}) {
	config.C.SetConfigFile(filepath.Join(util.RootDir(), "data", "conf.toml"))
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			config.C.SetConfigFile(v)
		case ConfigFunc:
			v(config.C)
		case Callback:
			v()
		default:
			vv := reflect.ValueOf(v)
			if vv.Type().ConvertibleTo(mapType) {
				m1 := vv.Convert(mapType).Interface().(map[string]interface{})
				config.MergeGlobalDefaults(m1)
				continue
			}
			panic(fmt.Sprintf("LoadConfig unknown type %T", v))
		}
	}
	config.C.Load()
}

func RegisterAfterHttpServer(name string, fn func()) {
	afterHttpServerMap[name] = fn
}

func RegisterAfterInject(name string, fn func()) {
	afterInjectMap[name] = fn
}

func RegisterBeforeHttpServer(name string, fn func()) {
	beforeHttpServerMap[name] = fn
}

func RegisterBeforeInject(name string, fn func()) {
	beforeInjectMap[name] = fn
}

func RegisterShutdown(name string, fn func()) {
	shutdownMap[name] = fn
}

func Start() {
	StartupWithoutHttpServer()
	if isRunEmq() {
		gemq.API.Handle(giris.App.IrisApp.Party("/"))
	}
	giris.App.StartApp()
	for _, fn := range afterHttpServerMap {
		fn()
	}
	startEmq()
	giris.App.SysLogger.Println("app is running...")
	config.C.SetValue("app.runtimes", config.Viper().GetInt("app.runtimes")+1)
	iris.RegisterOnInterrupt(func() {
		log.Println("Server stopping...")
		for _, fn := range shutdownMap {
			fn()
		}
		giris.App.Shutdown()
	})
	giris.App.Wait()
	closeServiceConns()
}

func StartupWithoutHttpServer() {
	for _, fn := range beforeInjectMap {
		fn()
	}

	util.InjectPopulate(
		giris.App,
		config.C,
		giris.IncJWT,
		util.ValidatorInc,
		j2rpc.New(j2rpc.SnakeOption.SetService(ServiceApp)),
		util.ZapLogger("sys", config.Viper().GetString("app.log_level")),
	)

	for _, fn := range afterInjectMap {
		fn()
	}

	j2rpc.PopulateConstructor(giris.App, config.C.AppDebug())

	// init redis
	redisOpt := parseRedisOpt()
	if redisOpt != nil {
		mdb.Rdb.Initialize(redisOpt)
	}

	// init db
	dbOpt := parseDBOpt()
	if dbOpt != nil {
		mdb.DB.Initialize(dbOpt)
	}

	for _, fn := range beforeHttpServerMap {
		fn()
	}
}

func closeServiceConns() {
	giris.App.SysLogger.Println("app is shutdown...")
	if gemq.Inc != nil && gemq.Inc.Client() != nil {
		gemq.Inc.Client().Disconnect(100)
	}
	if mdb.Rdb != nil {
		mdb.Rdb.Close()
	}
	if mdb.DB.DB != nil {
		_ = mdb.DB.Close()
	}
	util.CloseWriters()
}

func isRunEmq() bool {
	port := config.Viper().GetString("emq.port")
	return port != ""
}

func parseDBOpt() (opt *mdb.DBOption) {
	dbConf := config.Viper().Sub("db")
	if dbConf == nil {
		return
	}
	dbType := dbConf.GetString("type")
	if dbType == "" {
		return
	}
	port := dbConf.GetString("port")
	if port == "" {
		return
	}
	dbName := dbConf.GetString("db")
	if dbName == "" {
		dbName = config.Viper().GetString("app.name")
	}
	host := dbConf.GetString("host")
	if host == "" {
		host = config.Viper().GetString("app.host")
	}
	logLevelVal := logger.LogLevel(dbConf.GetInt("log_level"))
	if logLevelVal == 0 {
		logLevelVal = logger.Error
	}
	return &mdb.DBOption{
		Type:               dbType,
		Host:               host,
		Port:               port,
		User:               dbConf.GetString("user"),
		Pwd:                dbConf.GetString("pwd"),
		DB:                 dbName,
		SkipCache:          dbConf.GetBool("skip_cache"),
		CacheType:          dbConf.GetString("cache_type"),
		Logger:             mdb.NewDBLoggerWithLevel(logLevelVal),
		MaxIdleConns:       dbConf.GetInt("max_idle_conns"),
		MaxOpenConns:       dbConf.GetInt("max_open_conns"),
		MaxLifetimeSeconds: dbConf.GetInt64("max_lifetime_seconds"),
		MaxIdleTimeSeconds: dbConf.GetInt64("max_idle_time_seconds"),
	}
}

func parseRedisOpt() (opt *mdb.RedisOptions) {
	redisConf := config.Viper().Sub("redis")
	if redisConf == nil {
		return
	}
	port := redisConf.GetString("port")
	if port == "" {
		return
	}
	host := redisConf.GetString("host")
	if host == "" {
		host = config.Viper().GetString("app.host")
	}
	return &mdb.RedisOptions{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: redisConf.GetString("pwd"),
		DB:       redisConf.GetInt("db"),
	}
}

func startEmq() {
	if !isRunEmq() {
		return
	}

	util.InjectPopulate(
		gemq.Inc,
		config.C,
		giris.App.SysLogger,
	)
	gemq.Inc.Constructor()
	gemq.Inc.Dial()
}
