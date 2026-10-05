package zapdecor_test

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	zapdecor "github.com/eggs-gd/go-zap-decor"
	"github.com/eggs-gd/go-zap-decor/tree"
)

var ansi = regexp.MustCompile("\033\\[[0-9;]*m")

// logged: what the logger wrote, without colors
func logged(buf *bytes.Buffer) string { return ansi.ReplaceAllString(buf.String(), "") }

// A disabled service is silent; enabled again, it logs
func TestDisableService(t *testing.T) {
	var buf bytes.Buffer
	log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{}, zapdecor.Output(&buf)).Named("importer")
	log.DisableService("importer")
	log.Info("hidden")
	log.EnableService("importer")
	log.Info("shown")
	if out := logged(&buf); strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Errorf("got %q", out)
	}
}

// Fields given with With go through the decorator too, with the entry's own
func TestWithDecorated(t *testing.T) {
	var buf bytes.Buffer
	log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{}, zapdecor.Output(&buf)).With(zapdecor.String("pass", "7"))
	log.Info("walked", zapdecor.Int("files", 3))
	if out := logged(&buf); !strings.Contains(out, "├─ pass: 7\n└─ files: 3\n") {
		t.Errorf("got %q", out)
	}
}

// Every field by its type: zero values kept, a duration as a duration
func TestFieldsByType(t *testing.T) {
	var buf bytes.Buffer
	log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{}, zapdecor.Output(&buf))
	log.Info("x", zapdecor.Int("count", 0), zapdecor.String("empty", ""), zapdecor.Duration("took", 1500*time.Millisecond))
	out := logged(&buf)
	for _, want := range []string{"count: 0", "empty: \n", "took: 1.5s"} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in %q", want, out)
		}
	}
}

// An error's fields come right under its line; the stack trace after them
func TestStackAfterFields(t *testing.T) {
	var buf bytes.Buffer
	log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{}, zapdecor.Output(&buf))
	log.Error("failed", zapdecor.String("path", "/a.jpg"), zapdecor.Error(errors.New("disk full")))
	out := logged(&buf)
	fields, stack := strings.Index(out, "path: /a.jpg"), strings.Index(out, ".TestStackAfterFields\n")
	if fields < 0 || stack < 0 || stack < fields {
		t.Errorf("fields at %d, stack at %d:\n%s", fields, stack, out)
	}
}

// Named from many goroutines while services are switched (run with -race)
func TestNamedConcurrently(t *testing.T) {
	var buf bytes.Buffer
	log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{}, zapdecor.Output(&buf))
	var all sync.WaitGroup
	for i := range 20 {
		all.Go(func() {
			name := string(rune('a' + i%5))
			log.Named(name).DisableService(name)
			log.Named(name).EnableService(name)
		})
	}
	all.Wait()
}
