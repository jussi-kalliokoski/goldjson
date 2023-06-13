// Package goldjson provides utilities for handling line-delimited JSON.
//
// The main use case of the package is to provide a performant base tool for
// writing custom log/slog Handlers.
package goldjson

import (
	"io"
	"sync"
	"time"

	"github.com/jussi-kalliokoski/goldjson/tokens"
)

// Encoder is used for encoding line-delimited JSON records.
type Encoder struct {
	keys keyStore
	w    io.Writer
	p    sync.Pool
}

// NewEncoder returns a new Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// PrepareKey caches the encoded version of a key to make it faster to encode.
//
// NOTE: Not thread-safe, MUST only be called before using the Encoder.
func (e *Encoder) PrepareKey(key string) {
	e.keys.Put(key)
}

// NewLine creates a new line to be written to the writer.
func (e *Encoder) NewLine() *LineWriter {
	l, _ := e.p.Get().(*LineWriter)
	if l == nil {
		l = &LineWriter{encoder: e}
	}
	l.buf = append(l.buf, '{')
	l.isFirstEntry = 1
	return l
}

// LineWriter represents a line-delimited JSON record/list.
type LineWriter struct {
	buf          []byte
	depth        int
	isFirstEntry uint64
	isArray      uint64
	parent       *LineWriter
	encoder      *Encoder
}

// End finishes the line and writes it to the underlying writer of the Encoder.
//
// To get well-formed JSON, the caller MUST ensure that all inner records have
// been ended.
//
// After calling End, the LineWriter can no longer be used.
//
// Returns the error from the underlying writer, if any.
func (l *LineWriter) End() error {
	l.buf = append(l.buf, '}', '\n')
	_, err := l.encoder.w.Write(l.buf)
	l.buf = l.buf[:0]
	l.encoder.p.Put(l)
	return err
}

// AddString adds a key-value pair with a string value to the active
// record/list.
//
// If a list is currently active, the key will be ignored.
func (l *LineWriter) AddString(key, value string) {
	l.appendKey(key)
	l.buf = tokens.AppendString(l.buf, value)
}

// AddInt64 adds a key-value pair with an int64 value to the active
// record/list.
//
// If a list is currently active, the key will be ignored.
func (l *LineWriter) AddInt64(key string, value int64) {
	l.appendKey(key)
	l.buf = tokens.AppendInt64(l.buf, value)
}

// AddUint64 adds a key-value pair with a uint64 value to the active
// record/list.
//
// If a list is currently active, the key will be ignored.
func (l *LineWriter) AddUint64(key string, value uint64) {
	l.appendKey(key)
	l.buf = tokens.AppendUint64(l.buf, value)
}

// AddBool adds a key-value pair with a bool value to the active record/list.
//
// If a list is currently active, the key will be ignored.
func (l *LineWriter) AddBool(key string, value bool) {
	l.appendKey(key)
	l.buf = tokens.AppendBool(l.buf, value)
}

// AddFloat64 adds a key-value pair with a float64 value to the active
// record/list.
//
// If a list is currently active, the key will be ignored.
func (l *LineWriter) AddFloat64(key string, value float64) {
	l.appendKey(key)
	l.buf = tokens.AppendFloat64(l.buf, value)
}

// AddTime adds a key-value pair with a time.Time value to the active
// record/list.
func (l *LineWriter) AddTime(key string, value time.Time) error {
	orig := l.buf
	l.appendKey(key)
	var err error
	l.buf, err = tokens.AppendTime(l.buf, value)
	if err != nil {
		l.buf = orig
		return err
	}
	return err
}

// AddMarshal adds a key-value pair with a JSON value to the active
// record/list.
func (l *LineWriter) AddMarshal(key string, value any) error {
	orig := l.buf
	l.appendKey(key)
	var err error
	l.buf, err = tokens.AppendMarshal(l.buf, value)
	if err != nil {
		l.buf = orig
		return err
	}
	return nil
}

// StartRecord creates a new key-value pair to the active record/list with a
// record type.
//
// If a list is currently active, the key will be ignored.
//
// EndRecord MUST be called after all the pairs of the record have been added.
func (l *LineWriter) StartRecord(key string) {
	l.appendKey(key)
	l.buf = append(l.buf, '{')
	if l.depth == 63 {
		parent := &LineWriter{}
		*parent = *l
		*l = LineWriter{
			buf:          l.buf,
			isFirstEntry: 1,
			depth:        0,
			parent:       parent,
			encoder:      l.encoder,
		}
		return
	}
	l.depth++
	l.isFirstEntry = l.isFirstEntry | (1 << l.depth)
	l.isArray = l.isArray & ^(1 << l.depth)
}

// EndRecord closes the active record.
//
// If the active record is the top-level record, this function will panic.
func (l *LineWriter) EndRecord() {
	l.depth--
	if l.depth == -1 {
		parent := l.parent
		parent.buf = l.buf
		*l = *parent
	}
	l.buf = append(l.buf, '}')
}

// StartList creates a new key-value pair to the active record with a list
// type.
//
// EndList MUST be called after all the values of the list have been added.
func (l *LineWriter) StartList(key string) {
	l.appendKey(key)
	l.buf = append(l.buf, '[')
	if l.depth == 63 {
		parent := &LineWriter{}
		*parent = *l
		*l = LineWriter{
			buf:          l.buf,
			isFirstEntry: 1,
			isArray:      1,
			depth:        0,
			parent:       parent,
		}
		return
	}
	l.depth++
	l.isFirstEntry = l.isFirstEntry | (1 << l.depth)
	l.isArray = l.isArray | (1 << l.depth)
}

// EndList closes the active list.
func (l *LineWriter) EndList() {
	l.depth--
	if l.depth == -1 {
		parent := l.parent
		parent.buf = l.buf
		*l = *parent
	}
	l.buf = append(l.buf, ']')
}

func (l *LineWriter) appendKey(key string) {
	if l.isFirstEntry&(1<<l.depth) == 0 {
		l.buf = append(l.buf, ',')
	} else {
		l.isFirstEntry = l.isFirstEntry ^ (1 << l.depth)
	}
	if l.isArray&(1<<l.depth) == 0 {
		l.buf = l.encoder.keys.Append(l.buf, key)
		l.buf = append(l.buf, ':')
	}
}
