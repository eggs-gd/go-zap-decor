package zapdecor

import (
	"io"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ServiceConfig holds configuration for a service logger
type ServiceConfig struct {
	Name    string
	Color   string
	Enabled bool
}

// Logger wraps zap logger with additional service management
type Logger struct {
	zapLogger *zap.Logger
	service   string // its own name (Named), plain: the key of its ServiceConfig
	services  map[string]ServiceConfig
	mu        *sync.RWMutex
}

type LogLevel int8

// Option: a zap option, or where the logger writes (Output)
type Option struct {
	zapOption zap.Option
	out       io.Writer
}

// Output: the logger writes here instead of stdout
func Output(w io.Writer) Option { return Option{out: w} }

const (
	DebugLevel  LogLevel = LogLevel(zapcore.DebugLevel)
	InfoLevel   LogLevel = LogLevel(zapcore.InfoLevel)
	WarnLevel   LogLevel = LogLevel(zapcore.WarnLevel)
	ErrorLevel  LogLevel = LogLevel(zapcore.ErrorLevel)
	DPanicLevel LogLevel = LogLevel(zapcore.DPanicLevel)
	PanicLevel  LogLevel = LogLevel(zapcore.PanicLevel)
	FatalLevel  LogLevel = LogLevel(zapcore.FatalLevel)
)

const (
	ColorBlack   = "\033[30m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"

	ColorBrightBlack   = "\033[90m"
	ColorBrightRed     = "\033[91m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightBlue    = "\033[94m"
	ColorBrightMagenta = "\033[95m"
	ColorBrightCyan    = "\033[96m"
	ColorBrightWhite   = "\033[97m"

	ColorBgBlack   = "\033[40m"
	ColorBgRed     = "\033[41m"
	ColorBgGreen   = "\033[42m"
	ColorBgYellow  = "\033[43m"
	ColorBgBlue    = "\033[44m"
	ColorBgMagenta = "\033[45m"
	ColorBgCyan    = "\033[46m"
	ColorBgWhite   = "\033[47m"

	ColorReset = "\033[0m"
)
