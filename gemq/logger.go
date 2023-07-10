package gemq

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/atcharles/glibs/util"
)

func newInnerLogger(logger util.ItfLogger) *innerLogger {
	return &innerLogger{Logger: logger}
}

type innerLogger struct {
	Logger util.ItfLogger
}

// WarnLogger ...
func (i *innerLogger) WarnLogger() mqtt.Logger {
	return util.ZapLogger("sys", "warn", "mqtt")
}

// CriticalLogger ...
func (i *innerLogger) CriticalLogger() mqtt.Logger {
	return util.ZapLogger("sys", "warn", "mqtt")
}

// ErrorLogger ...
func (i *innerLogger) ErrorLogger() mqtt.Logger {
	return util.ZapLogger("sys", "error", "mqtt")
}
