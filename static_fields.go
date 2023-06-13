package goldjson

// StaticFields represents a pre-built set of record fields.
type StaticFields struct {
	buf []byte
}

// NewStaticFields can be used for caching static fields in a record for
// better performance. Returns a StaticFields and a LineWriter to populate
// it.
//
// Use End() on the LineWriter to complete the StaticFields construction.
func NewStaticFields() (*StaticFields, *LineWriter) {
	f := &StaticFields{}
	l := &LineWriter{
		isFirstEntry: 1,
		encoder:      &Encoder{w: staticFieldsWriter{f}},
	}

	return f, l
}

type staticFieldsWriter struct {
	staticFields *StaticFields
}

func (w staticFieldsWriter) Write(data []byte) (n int, err error) {
	// trim trailing closing brace and newline
	w.staticFields.buf = data[:len(data)-2]
	return len(data), nil
}
