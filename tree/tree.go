package tree

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"

	zapdecor "github.com/eggs-gd/go-zap-decor"
)

const (
	TreeBranch     = "├─"
	TreeLastBranch = "└─"
	TreePipe       = "│ "
)

// Decorator: an entry's fields as a tree under its line, an error last in red; a
// GORM-style SQL entry (fields sql, rows, elapsed) as the rows and the time, then
// the statement with its keywords, strings and numbers colored
type Decorator struct{}

func (c *Decorator) Decorate(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	fields = cleanupFields(fields)
	if len(fields) == 0 {
		return buf
	}

	if hasField(fields, "sql") {
		buf = c.decorateSQL(buf, fields)
	} else {
		buf = c.decorateDefault(buf, fields)
	}

	if hasField(fields, "error") {
		buf = c.decorateError(buf, fields)
	}

	buf.AppendString("\n")
	return buf
}

func (c *Decorator) decorateSQL(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	var elapsed, rows, sql string

	for _, field := range fields {
		switch field.Key {
		case "elapsed":
			if field.Integer != 0 {
				elapsed = time.Duration(field.Integer).String()
			}
		case "rows":
			if field.Integer >= 0 { // GORM reports -1 when rows are not applicable
				rows = fmt.Sprintf("%d", field.Integer)
			}
		case "sql":
			if field.String != "" {
				sql = field.String
			}
		}
	}

	if rows != "" || elapsed != "" {
		buf.AppendString(fmt.Sprintf("%s \033[36m%v rows in %v\033[0m\n", TreeBranch, rows, elapsed))
	}
	if sql != "" {
		buf.AppendString(fmt.Sprintf("%s %v\n", TreeLastBranch, formatSQL(sql)))
	}
	return buf
}

func (c *Decorator) decorateError(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	for _, field := range fields {
		if field.Key == "error" {
			buf.AppendString(fmt.Sprintf("\033[31m╳ %v\033[0m\n", field.Interface))
			return buf
		}
	}
	return buf
}

func (c *Decorator) decorateDefault(buf *buffer.Buffer, fields []zapcore.Field) *buffer.Buffer {
	lastIdx := len(fields) - 1

	for i, field := range fields {
		if field.Key == "error" {
			continue
		}
		prefix := TreeBranch
		if i == lastIdx {
			prefix = TreeLastBranch
		}
		buf.AppendString(fmt.Sprintf("%s %s: %s\n", prefix, field.Key, formatFieldValue(field)))
	}
	return buf
}

// formatFieldValue: the field's value by its type, as zap would encode it — a zero
// is a value, a duration a duration
func formatFieldValue(field zapcore.Field) string {
	enc := zapcore.NewMapObjectEncoder()
	field.AddTo(enc)
	return fmt.Sprint(enc.Fields[field.Key])
}

func cleanupFields(fields []zapcore.Field) []zapcore.Field {
	validFields := []zapcore.Field{}
	for _, field := range fields {
		if field.Key != "" {
			validFields = append(validFields, field)
		}
	}
	return validFields
}

func hasField(fields []zapcore.Field, key string) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

var (
	// Longest alternatives first so "GROUP BY" wins over "BY"-less matches.
	sqlKeywordRe = regexp.MustCompile(`(?i)\b(GROUP BY|ORDER BY|RETURNING|CONFLICT|SELECT|INSERT|VALUES|UPDATE|DELETE|HAVING|OFFSET|OUTER|INNER|RIGHT|WHERE|LIMIT|INTO|FROM|JOIN|LEFT|NULL|AND|NOT|SET|AS|DO|IN|IS|ON|OR)\b`)
	sqlStringRe  = regexp.MustCompile(`'[^']*'`)
	sqlNumberRe  = regexp.MustCompile(`,\b(\d+\.?\d*)\b,`)
	sqlGuidRe    = regexp.MustCompile(`"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"`)
)

func formatSQL(sql string) string {
	sql = strings.TrimSpace(sql)
	sql = strings.Join(strings.Fields(sql), " ")

	sql = sqlKeywordRe.ReplaceAllStringFunc(sql, func(s string) string {
		return zapdecor.ColorBrightCyan + strings.ToUpper(s) + zapdecor.ColorReset
	})
	sql = sqlStringRe.ReplaceAllStringFunc(sql, func(s string) string {
		return zapdecor.ColorGreen + s + zapdecor.ColorReset
	})
	sql = sqlNumberRe.ReplaceAllString(sql, ","+zapdecor.ColorYellow+"${1}"+zapdecor.ColorReset+",")
	sql = sqlGuidRe.ReplaceAllStringFunc(sql, func(s string) string {
		return zapdecor.ColorBrightMagenta + s + zapdecor.ColorReset
	})

	return sql
}
