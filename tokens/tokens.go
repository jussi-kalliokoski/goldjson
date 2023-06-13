package tokens

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

// AppendInt64 appends an encoded int64 value to the buffer.
func AppendInt64(buf []byte, value int64) []byte {
	return strconv.AppendInt(buf, value, 10)
}

// AppendUint64 appends an encoded uint64 value to the buffer.
func AppendUint64(buf []byte, value uint64) []byte {
	return strconv.AppendUint(buf, value, 10)
}

// AppendBool appends an encoded bool value to the buffer.
func AppendBool(buf []byte, value bool) []byte {
	return strconv.AppendBool(buf, value)
}

// AppendFloat64 appends an encoded float64 value to the buffer.
//
// Unline json.Marshal, positive/negative Infinity values and NaNs are
// marshaled as strings ("+Inf", "-Inf", "NaN" respectively) instead of
// erroring.
func AppendFloat64(buf []byte, value float64) []byte {
	// json.Marshal fails on special floats, so handle them here.
	switch {
	case math.IsInf(value, 1):
		return fmt.Append(buf, "\"+Inf\"")
	case math.IsInf(value, -1):
		return fmt.Append(buf, "\"-Inf\"")
	case math.IsNaN(value):
		return fmt.Append(buf, "\"NaN\"")
	default:
		abs := math.Abs(value)
		fmt := byte('f')
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
		oldLen := len(buf)
		buf = strconv.AppendFloat(buf, value, fmt, -1, 64)
		b := buf[oldLen:]
		if fmt == 'e' {
			// clean up e-09 to e-9
			if n := len(b); n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
				b[n-2] = b[n-1]
				buf = buf[:len(buf)-1]
			}
		}
		return buf
	}
}

// AppendTime appends an encoded time value to the buffer.
func AppendTime(buf []byte, value time.Time) ([]byte, error) {
	if y := value.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		return buf, errors.New("time.Time year outside of range [0,9999]")
	}
	buf = append(buf, '"')
	buf = value.AppendFormat(buf, time.RFC3339Nano)
	return append(buf, '"'), nil
}

// AppendMarshal appends an encoded JSON value to the buffer.
func AppendMarshal(buf []byte, value any) ([]byte, error) {
	bw := bytesWriter{buf}
	enc := json.NewEncoder(&bw)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return buf, err
	}
	buf = bw.buf[:len(bw.buf)-1] // remove final newline
	return buf, nil
}

// AppendString appends an encoded (quoted and escaped) string value to the
// buffer.
func AppendString(buf []byte, s string) []byte {
	buf = append(buf, '"')
	buf = appendJSONString(buf, s)
	return append(buf, '"')
}

// appendJSONString escapes s for JSON and appends it to buf.
// It does not surround the string in quotation marks.
//
// Modified from encoding/json/encode.go:encodeState.string,
// with escapeHTML set to false.
func appendJSONString(buf []byte, s string) []byte {
	char := func(b byte) { buf = append(buf, b) }
	str := func(s string) { buf = append(buf, s...) }

	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if safeSet[b] {
				i++
				continue
			}
			if start < i {
				str(s[start:i])
			}
			char('\\')
			switch b {
			case '\\', '"':
				char(b)
			case '\n':
				char('n')
			case '\r':
				char('r')
			case '\t':
				char('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// It also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				str(`u00`)
				char(hex[b>>4])
				char(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				str(s[start:i])
			}
			str(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				str(s[start:i])
			}
			str(`\u202`)
			char(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		str(s[start:])
	}
	return buf
}

var hex = "0123456789abcdef"

// Copied from encoding/json/tables.go.
//
// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

type bytesWriter struct {
	buf []byte
}

func (w *bytesWriter) Write(data []byte) (n int, err error) {
	w.buf = append(w.buf, data...)
	return len(data), nil
}
