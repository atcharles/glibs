package giris

import (
	"github.com/kataras/iris/v12/middleware/accesslog"

	"github.com/atcharles/glibs/config"
	"github.com/atcharles/glibs/util"
)

type midLogger struct {
	logger    *accesslog.AccessLog
	webLogger util.ItfLogger
}

func (m *midLogger) Constructor() {
	m.webLogger = util.ZapLogger("web", config.Viper().GetString("app.log_level"))
}

// Logger ......
func (m *midLogger) Logger() *accesslog.AccessLog {
	ac := accesslog.New(m.WebLogger().Writer())
	// The default configuration:
	ac.Delim = '|'
	ac.TimeFormat = "2006-01-02 15:04:05.000"
	ac.Async = false
	ac.IP = true
	ac.BytesReceivedBody = true
	ac.BytesSentBody = true
	ac.BytesReceived = true
	ac.BytesSent = true
	ac.BodyMinify = true
	ac.RequestBody = true
	ac.ResponseBody = false
	ac.KeepMultiLineError = true
	ac.PanicLog = accesslog.LogHandler

	// Default line format if formatter is missing:
	// Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
	//
	// Set Custom Formatter:
	//ac.SetFormatter(&accesslog.JSON{Indent: "", HumanTime: true})
	// ac.SetFormatter(&accesslog.CSV{})
	const defaultTmplText = "{{.Now.Format .TimeFormat}} | {{.Latency}}" +
		" | {{.Code}} | {{.Method}} | {{.Path}} | {{.IP}}" +
		" | {{.RequestValuesLine}} | {{.BytesReceivedLine}} | {{.BytesSentLine}}" +
		" | {{.Request}} | {{.Response}} |\n"
	ac.SetFormatter(&accesslog.Template{Text: defaultTmplText})

	m.logger = ac
	return m.logger
}

func (m *midLogger) WebLogger() util.ItfLogger {
	return m.webLogger
}
