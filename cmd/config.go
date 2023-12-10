package cmd

import (
	"github.com/atcharles/glibs/config"
)

var GetConfigFunc = func() *Config {
	return &Config{
		APPName:         config.C.AppName(),
		ServerHost:      config.Viper().GetString("server.host"),
		ServerPProfPort: config.Viper().GetString("server.pprof_port"),
	}
}

type Config struct {
	APPName         string
	ServerHost      string
	ServerPProfPort string
}
