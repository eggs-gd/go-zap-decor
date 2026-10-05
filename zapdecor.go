// Package zapdecor: a colored console logger on zap whose entry fields are written
// by a decorator of your choice. Each line is the time, the colored level, the
// logger's name (each named service in its own color) and the message; then the
// Decorator writes the fields — tree.Decorator as a tree, or your own.
package zapdecor

import (
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Decorator: how an entry's fields are written after its line — the console
// encoder writes the time, the level, the logger's name and the message, then
// hands the fields to the decorator (tree.Decorator is one: fields as a tree)
type Decorator interface {
	Decorate(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer
}

// encoder: zap's console encoder for the line, the decorator for the fields, then
// the stack trace — after the fields, not between the line and them
type encoder struct {
	zapcore.Encoder
	decorator Decorator
}

// Clone keeps the decorator (zap clones the encoder for a logger's own fields)
func (e *encoder) Clone() zapcore.Encoder {
	return &encoder{Encoder: e.Encoder.Clone(), decorator: e.decorator}
}

func (e *encoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	stack := entry.Stack
	entry.Stack = ""
	buf, err := e.Encoder.EncodeEntry(entry, nil)
	if err != nil {
		return nil, err
	}
	buf = e.decorator.Decorate(buf, fields)
	if stack != "" {
		buf.AppendString(stack)
		buf.AppendString("\n")
	}
	return buf, nil
}

func newEncoder(decorator Decorator) *encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("15:04:05.000"))
	}
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	encoderConfig.NewReflectedEncoder = func(w io.Writer) zapcore.ReflectedEncoder {
		return &consoleEncoder{w: w}
	}
	return &encoder{Encoder: zapcore.NewConsoleEncoder(encoderConfig), decorator: decorator}
}

// core: keeps a logger's own fields (With) itself and hands them to the encoder
// with each entry's — so they reach the decorator too, instead of being written
// inline by zap's encoder
type core struct {
	zapcore.Core
	fields []zapcore.Field
}

func (c *core) With(fields []zapcore.Field) zapcore.Core {
	return &core{Core: c.Core, fields: append(slices.Clone(c.fields), fields...)}
}

func (c *core) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(entry, append(slices.Clone(c.fields), fields...))
}

// NewLogger: a colored console logger at this level (to stdout, or Output), its
// fields written by the decorator; Named gives each service its own color
func NewLogger(level LogLevel, decorator Decorator, options ...Option) *Logger {
	var out io.Writer = os.Stdout
	for _, o := range options {
		if o.out != nil {
			out = o.out
		}
	}
	core := &core{Core: zapcore.NewCore(newEncoder(decorator), zapcore.Lock(zapcore.AddSync(out)), zapcore.Level(level))}

	stackLevels := []zapcore.Level{
		zapcore.ErrorLevel,
		zapcore.DPanicLevel,
		zapcore.PanicLevel,
		zapcore.FatalLevel,
	}

	defaultOptions := []zap.Option{
		zap.AddStacktrace(zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			for _, stackLevel := range stackLevels {
				if lvl >= stackLevel {
					return true
				}
			}
			return false
		})),
	}

	logger := &Logger{
		zapLogger: zap.New(core, append(defaultOptions, convertOptions(options)...)...),
		services:  make(map[string]ServiceConfig),
		mu:        &sync.RWMutex{},
	}
	return logger
}

func (l *Logger) log(level LogLevel, msg string, fields ...zap.Field) {
	if l.service != "" && !l.isServiceEnabled(l.service) {
		return
	}

	switch level {
	case DebugLevel:
		l.zapLogger.Debug(msg, fields...)
	case InfoLevel:
		l.zapLogger.Info(msg, fields...)
	case WarnLevel:
		l.zapLogger.Warn(msg, fields...)
	case ErrorLevel:
		l.zapLogger.Error(msg, fields...)
	case DPanicLevel:
		l.zapLogger.DPanic(msg, fields...)
	case PanicLevel:
		l.zapLogger.Panic(msg, fields...)
	case FatalLevel:
		l.zapLogger.Fatal(msg, fields...)
	}
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.log(DebugLevel, msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.log(InfoLevel, msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.log(WarnLevel, msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.log(ErrorLevel, msg, fields...)
}

func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	l.log(DPanicLevel, msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.log(PanicLevel, msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.log(FatalLevel, msg, fields...)
}

func (l *Logger) Named(name string) *Logger {
	color := getColorForService(name)
	l.mu.RLock()
	_, exists := l.services[name]
	l.mu.RUnlock()
	if !exists {
		l.RegisterService(name, color)
	}

	namedLogger := l.zapLogger.Named(fmt.Sprintf("%s%s%s", color, name, ColorReset))
	return &Logger{
		zapLogger: namedLogger,
		service:   name,
		services:  l.services,
		mu:        l.mu,
	}
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		zapLogger: l.zapLogger.With(fields...),
		service:   l.service,
		services:  l.services,
		mu:        l.mu,
	}
}

func (l *Logger) WithOptions(options ...Option) *Logger {
	zapOptions := convertOptions(options)
	return &Logger{
		zapLogger: l.zapLogger.WithOptions(zapOptions...),
		service:   l.service,
		services:  l.services,
		mu:        l.mu,
	}
}

func convertOptions(options []Option) []zap.Option {
	var zapOptions []zap.Option
	for _, opt := range options {
		if opt.zapOption != nil {
			zapOptions = append(zapOptions, opt.zapOption)
		}
	}
	return zapOptions
}

type consoleEncoder struct {
	w io.Writer
}

func (e *consoleEncoder) Encode(v interface{}) error {
	_, err := fmt.Fprintf(e.w, "%+v", v)
	return err
}
