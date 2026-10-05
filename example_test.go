package zapdecor_test

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"

	zapdecor "github.com/eggs-gd/go-zap-decor"
	"github.com/eggs-gd/go-zap-decor/tree"
)

// A logger whose fields come out as a tree; each named service gets its color.
//
//	15:04:05.000  INFO  importer  Walk complete
//	├─ files: 592
//	└─ unreadable: 1
func Example() {
	log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{})
	importer := log.Named("importer")
	importer.Info("Walk complete", zapdecor.Int("files", 592), zapdecor.Int("unreadable", 1))
	importer.Error("File not stored", zapdecor.String("path", "/photos/a.jpg"), zapdecor.Error(errors.New("disk full")))
	importer.Debug("not shown: below the level", zapdecor.Duration("took", time.Second))
}

// inline: a decorator of your own — fields as key=value on the same line
type inline struct{}

func (inline) Decorate(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	for _, f := range fields {
		buf.AppendString(" " + f.Key + "=")
		switch {
		case f.String != "":
			buf.AppendString(f.String)
		case f.Interface != nil:
			buf.AppendString(fmt.Sprint(f.Interface))
		default:
			buf.AppendInt(f.Integer)
		}
	}
	buf.AppendString("\n")
	return buf
}

// A decorator of your own: one method, given zap's buffer with the line in it.
func ExampleDecorator() {
	log := zapdecor.NewLogger(zapdecor.DebugLevel, inline{})
	log.Named("render").Info("Rendered", zapdecor.Int("photos", 64), zapdecor.Duration("took", 2*time.Second))
}
