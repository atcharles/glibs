package j2rpc

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// level
const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

// Level type
type Level uint32

// MarshalText ...
func (l *Level) MarshalText() ([]byte, error) {
	switch *l {
	case TraceLevel:
		return []byte("trace"), nil
	case DebugLevel:
		return []byte("debug"), nil
	case InfoLevel:
		return []byte("info"), nil
	case WarnLevel:
		return []byte("warning"), nil
	case ErrorLevel:
		return []byte("error"), nil
	case FatalLevel:
		return []byte("fatal"), nil
	case PanicLevel:
		return []byte("panic"), nil
	}
	return []byte("debug"), nil
}

// Convert the Level to a string. E.g. PanicLevel becomes "panic".
func (l *Level) String() string {
	if b, err := l.MarshalText(); err == nil {
		return string(b)
	}
	return "unknown"
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *Level) UnmarshalText(text []byte) error {
	*l = ParseLevel(string(text))
	return nil
}

// LevelLogger ...
type LevelLogger interface {
	SetOutput(io.Writer)
	Writer() io.Writer

	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})

	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Fatalln(...interface{})

	Panic(...interface{})
	Panicf(string, ...interface{})
	Panicln(...interface{})

	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

type logger struct {
	lvl Level
	lg  *log.Logger
}

func (l *logger) Debugf(s string, i ...interface{}) {
	if l.lvl < DebugLevel {
		return
	}
	l.out(DebugLevel, s, i...)
}

func (l *logger) Errorf(s string, i ...interface{}) {
	if l.lvl < ErrorLevel {
		return
	}
	l.out(ErrorLevel, s, i...)
}

func (l *logger) Fatal(i ...interface{}) {
	if l.lvl < FatalLevel {
		return
	}
	l.lg.Fatal(i...)
}

func (l *logger) Fatalf(s string, i ...interface{}) {
	if l.lvl < FatalLevel {
		return
	}
	l.lg.Fatalf(s, i...)
}

func (l *logger) Fatalln(i ...interface{}) {
	if l.lvl < FatalLevel {
		return
	}
	l.lg.Fatalln(i...)
}

func (l *logger) Infof(s string, i ...interface{}) {
	if l.lvl < InfoLevel {
		return
	}
	l.out(InfoLevel, s, i...)
}

func (l *logger) Panic(i ...interface{}) {
	if l.lvl < PanicLevel {
		return
	}
	l.lg.Panic(i...)
}

func (l *logger) Panicf(s string, i ...interface{}) {
	if l.lvl < PanicLevel {
		return
	}
	l.lg.Panicf(s, i...)
}

func (l *logger) Panicln(i ...interface{}) {
	if l.lvl < PanicLevel {
		return
	}
	l.lg.Panicln(i...)
}

func (l *logger) Print(i ...interface{}) {
	if l.lvl < InfoLevel {
		return
	}
	l.lg.Print(i...)
}

func (l *logger) Printf(s string, i ...interface{}) { l.Infof(s, i...) }

func (l *logger) Println(i ...interface{}) { l.lg.Println(i...) }

// SetLevel ...
func (l *logger) SetLevel(lvl Level) { l.lvl = lvl }

func (l *logger) SetOutput(writer io.Writer) { l.lg.SetOutput(writer) }

func (l *logger) Warnf(s string, i ...interface{}) {
	if l.lvl < WarnLevel {
		return
	}
	l.out(WarnLevel, s, i...)
}

func (l *logger) Writer() io.Writer { return l.lg.Writer() }

// out ...
func (l *logger) out(lvl Level, s string, i ...interface{}) {
	f1 := fmt.Sprintf("[%s] %s", strings.ToUpper(lvl.String()), s)
	l.lg.Printf(f1, i...)
}

// NewLevelLogger ...
func NewLevelLogger(prefix string, out ...io.Writer) LevelLogger {
	var oo io.Writer = os.Stdout
	if len(out) > 0 {
		oo = out[0]
	}
	lg := log.New(oo, fmt.Sprintf("[L]%s ", prefix), log.Ldate|log.Lmicroseconds)
	return &logger{lvl: DebugLevel, lg: lg}
}

// ParseLevel takes a string level and returns the log level constant.
func ParseLevel(lvl string) Level {
	switch strings.ToLower(lvl) {
	case "panic":
		return PanicLevel
	case "fatal":
		return FatalLevel
	case "error":
		return ErrorLevel
	case "warn", "warning":
		return WarnLevel
	case "info":
		return InfoLevel
	case "debug":
		return DebugLevel
	case "trace":
		return TraceLevel
	default:
		return DebugLevel
	}
}
