package log

import (
	"bytes"
	"fmt"
	"github.com/go-stack/stack"
	"github.com/inconshreveable/log15"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	timeFormat     = "2006-01-02T15:04:05-0700"
	termTimeFormat = "01-02|15:04:05"
	floatFormat    = 'f'
	termMsgJust    = 40
	errorKey       = "LOG15_ERROR"
)

// locationTrims are trimmed for display to avoid unwieldy log lines.
var locationTrims = []string{
	"github.com/LemoFoundationLtd/",
}

// locationLength is the maxmimum path length encountered, which all logs are
// padded to to aid in alignment.
var locationLength uint32

// fieldPadding is a global map with maximum field value lengths seen until now
// to allow padding log contexts in a bit smarter way.
var fieldPadding = make(map[string]int)

// fieldPaddingLock is a global mutex protecting the field padding map.
var fieldPaddingLock sync.RWMutex

// TerminalStringer is an analogous interface to the stdlib stringer, allowing
// own types to have custom shortened serialization formats when printed to the
// screen.
type TerminalStringer interface {
	TerminalString() string
}

// TerminalFormat formats log records optimized for human readability on
// a terminal with color-coded level output and terser human friendly timestamp.
// This format should only be used for interactive programs or while developing.
//
//     [TIME] [LEVEL] MESAGE key=value key=value ...
//
// Example:
//
//     [May 16 20:58:45] [DBUG] remove route ns=haproxy addr=127.0.0.1:50002
//
func TerminalFormat(usecolor bool, showCodeLine bool) log15.Format {
	return log15.FormatFunc(func(r *log15.Record) []byte {
		var color = 0
		if usecolor {
			switch r.Lvl {
			case log15.LvlCrit:
				color = 35
			case log15.LvlError:
				color = 31
			case log15.LvlWarn:
				color = 33
			case log15.LvlInfo:
				color = 32
			case log15.LvlDebug:
				color = 36
			}
		}

		b := &bytes.Buffer{}
		lvl := getAlignedString(r.Lvl)
		if showCodeLine {
			// Log origin printing was requested, format the location path and line number
			location := fmt.Sprintf("%+v", stack.Caller(16))
			for _, prefix := range locationTrims {
				location = strings.TrimPrefix(location, prefix)
			}
			// Maintain the maximum location length for fancyer alignment
			align := int(atomic.LoadUint32(&locationLength))
			if align < len(location) {
				align = len(location)
				atomic.StoreUint32(&locationLength, uint32(align))
			}
			padding := strings.Repeat(" ", align-len(location))

			// Assemble and print the log heading
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s|%s]%s %s ", color, lvl, r.Time.Format(termTimeFormat), location, padding, r.Msg)
			} else {
				fmt.Fprintf(b, "%s[%s|%s]%s %s ", lvl, r.Time.Format(termTimeFormat), location, padding, r.Msg)
			}
		} else {
			if color > 0 {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %s ", color, lvl, r.Time.Format(termTimeFormat), r.Msg)
			} else {
				fmt.Fprintf(b, "%s[%s] %s ", lvl, r.Time.Format(termTimeFormat), r.Msg)
			}
		}
		// try to justify the log output for short messages
		length := utf8.RuneCountInString(r.Msg)
		if len(r.Ctx) > 0 && length < termMsgJust {
			b.Write(bytes.Repeat([]byte{' '}, termMsgJust-length))
		}
		// print the keys logfmt style
		logfmt(b, r.Ctx, color, true)
		return b.Bytes()
	})
}

// Aligned returns a 5-character string containing the name of a Level.
func getAlignedString(l log15.Lvl) string {
	switch l {
	case log15.LvlDebug:
		return "DEBUG"
	case log15.LvlInfo:
		return "INFO "
	case log15.LvlWarn:
		return "WARN "
	case log15.LvlError:
		return "ERROR"
	case log15.LvlCrit:
		return "CRIT "
	default:
		panic("bad level")
	}
}

func logfmt(buf *bytes.Buffer, ctx []interface{}, color int, term bool) {
	for i := 0; i < len(ctx); i += 2 {
		if i != 0 {
			buf.WriteByte(' ')
		}

		k, ok := ctx[i].(string)
		v := formatLogfmtValue(ctx[i+1], term)
		if !ok {
			k, v = errorKey, formatLogfmtValue(k, term)
		}

		// XXX: we should probably check that all of your key bytes aren't invalid
		fieldPaddingLock.RLock()
		padding := fieldPadding[k]
		fieldPaddingLock.RUnlock()

		length := utf8.RuneCountInString(v)
		if padding < length {
			padding = length

			fieldPaddingLock.Lock()
			fieldPadding[k] = padding
			fieldPaddingLock.Unlock()
		}
		if color > 0 {
			fmt.Fprintf(buf, "\x1b[%dm%s\x1b[0m=", color, k)
		} else {
			buf.WriteString(k)
			buf.WriteByte('=')
		}
		buf.WriteString(v)
		if i < len(ctx)-2 {
			buf.Write(bytes.Repeat([]byte{' '}, padding-length))
		}
	}
	buf.WriteByte('\n')
}

func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return v.Format(timeFormat)

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}

// formatValue formats a value for serialization
func formatLogfmtValue(value interface{}, term bool) string {
	if value == nil {
		return "nil"
	}

	if t, ok := value.(time.Time); ok {
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return t.Format(timeFormat)
	}
	if term {
		if s, ok := value.(TerminalStringer); ok {
			// Custom terminal stringer provided, use that
			return escapeString(s.TerminalString())
		}
	}
	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 3, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 3, 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}

var stringBufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func escapeString(s string) string {
	needsQuotes := false
	needsEscape := false
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' {
			needsQuotes = true
		}
		if r == '\\' || r == '"' || r == '\n' || r == '\r' || r == '\t' {
			needsEscape = true
		}
	}
	if !needsEscape && !needsQuotes {
		return s
	}
	e := stringBufPool.Get().(*bytes.Buffer)
	e.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			e.WriteByte('\\')
			e.WriteByte(byte(r))
		case '\n':
			e.WriteString("\\n")
		case '\r':
			e.WriteString("\\r")
		case '\t':
			e.WriteString("\\t")
		default:
			e.WriteRune(r)
		}
	}
	e.WriteByte('"')
	var ret string
	if needsQuotes {
		ret = e.String()
	} else {
		ret = string(e.Bytes()[1 : e.Len()-1])
	}
	e.Reset()
	stringBufPool.Put(e)
	return ret
}
