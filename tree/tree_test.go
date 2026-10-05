package tree

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func decorate(fields ...zapcore.Field) string {
	buf := buffer.NewPool().Get()
	buf.AppendString("line\n")
	return (&Decorator{}).Decorate(buf, fields).String()
}

// Fields as a tree, the last one closing it; nothing to add when there are none
func TestTree(t *testing.T) {
	got := decorate(zap.String("path", "/a.jpg"), zap.Int("size", 42))
	want := "line\n├─ path: /a.jpg\n└─ size: 42\n\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := decorate(); got != "line\n" {
		t.Errorf("no fields: %q", got)
	}
}

// An error is written last, in red, not as a branch
func TestError(t *testing.T) {
	got := decorate(zap.String("path", "/a.jpg"), zap.Error(errors.New("disk full")))
	if !strings.Contains(got, "├─ path: /a.jpg") || !strings.HasSuffix(got, "\033[31m╳ disk full\033[0m\n\n") {
		t.Errorf("got %q", got)
	}
}

// An SQL entry: its rows and time, then the statement with its keywords colored
func TestSQL(t *testing.T) {
	got := decorate(zap.String("sql", "select * from items where id = 1"), zap.Int64("rows", 1), zap.Int64("elapsed", 1500000))
	if !strings.Contains(got, "1 rows in 1.5ms") || !strings.Contains(got, "SELECT") || !strings.Contains(got, "WHERE") {
		t.Errorf("got %q", got)
	}
}
