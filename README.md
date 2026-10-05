# go-zap-decor

[![Go Reference](https://pkg.go.dev/badge/github.com/eggs-gd/go-zap-decor.svg)](https://pkg.go.dev/github.com/eggs-gd/go-zap-decor)

**A decorated, colored console logger on [zap](https://github.com/uber-go/zap).**
Each entry is one line — the time, the colored level, the logger's name (every
named service in a color of its own) and the message — and below it, the entry's
fields written by a **decorator** you choose. One is included: fields as a tree.
Writing your own is a single method.

```
go get github.com/eggs-gd/go-zap-decor
```

```go
log := zapdecor.NewLogger(zapdecor.InfoLevel, &tree.Decorator{})
importer := log.Named("importer")

importer.Info("Walk complete", zapdecor.Int("files", 592), zapdecor.Int("unreadable", 1))
importer.Error("File not stored", zapdecor.String("path", "/photos/a.jpg"), zapdecor.Error(err))
```

```
15:09:30.162  INFO   importer  Walk complete
├─ files: 592
└─ unreadable: 1

15:09:31.020  ERROR  importer  File not stored
└─ path: /photos/a.jpg
╳ disk full
<stack trace>
```

(Colors in a terminal: the level, the service's name, errors in red.)

## The logger

- `NewLogger(level, decorator, options...)` — at `level` and above, to stdout or
  to `Output(w)`; a stack trace from `ErrorLevel` up, written after the fields.
- `Named(name)` — a child logger for a service: its name in a color of its own
  (picked once per name), joined with the parent's (`app.importer`).
- `With(fields...)` — fields of the logger itself: they go through the decorator
  too, before each entry's own. `WithOptions(...)` — as zap's.
- `DisableService(name)` / `EnableService(name)` — silence a named service at run
  time.
- Levels: `DebugLevel` … `FatalLevel`; field helpers: `String`, `Int`, `Int64`,
  `Duration`, `Error`, `Any` (zap's fields); ANSI color constants (`ColorCyan`, …).

## Decorators

The console encoder writes the line; the fields go to the decorator:

```go
type Decorator interface {
	Decorate(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer
}
```

It appends what it likes to `buf` (zap's buffer, already holding the line) and
returns it. Fields arrive as zap gives them (`Key`, `Type`, `Integer`, `String`,
`Interface`).

### `tree.Decorator`

- every field a branch (`├─`), the last one closing the tree (`└─`); a value by its
  zap type (a zero is a value, a duration a duration);
- an `error` field last, in red (`╳`);
- an SQL entry — fields `sql`, `rows`, `elapsed`, as GORM's logger gives them — as
  `n rows in t`, then the statement on one line with its keywords, strings and
  numbers colored.

### Your own

```go
// inline: fields as key=value on the same line
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

log := zapdecor.NewLogger(zapdecor.DebugLevel, inline{})
```

## License

MIT
