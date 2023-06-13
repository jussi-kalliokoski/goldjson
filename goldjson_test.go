package goldjson_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jussi-kalliokoski/goldjson"
)

func TestWidth(t *testing.T) {
	const maxSafeInteger = 9007199254740991
	const maxDepth = 1
	const maxPairsOfType = 2
	const maxStringSize = 16
	rng := rand.New(rand.NewSource(1))
	randomString := func(rng *rand.Rand, maxStringSize int) string {
		b := make([]byte, 1+rng.Intn(maxStringSize))
		_, _ = rng.Read(b)
		for i := range b {
			b[i] = b[i] & 0x7f
		}
		return string(b)
	}
	toRecord := func(keys []string, vals []any) map[string]any {
		r := make(map[string]any, len(keys))
		for i := range keys {
			r[keys[i]] = vals[i]
		}
		return r
	}
	var randomRecord func(line *goldjson.LineWriter, rng *rand.Rand, depth, maxDepth, maxPairsOfType, maxStringSize int) ([]string, []any)
	randomRecord = func(line *goldjson.LineWriter, rng *rand.Rand, depth, maxDepth, maxPairsOfType, maxStringSize int) ([]string, []any) {
		nBools := maxPairsOfType
		nInt64s := maxPairsOfType
		nUint64s := maxPairsOfType
		nFloat64s := maxPairsOfType
		nTimes := maxPairsOfType
		nStrings := maxPairsOfType
		nStructs := maxPairsOfType
		nRecords := maxPairsOfType
		nLists := maxPairsOfType
		if depth == maxDepth {
			nRecords = 0
			nLists = 0
		}
		nTotal := nBools + nInt64s + nUint64s + nFloat64s + nTimes + nStrings + nStructs + nRecords + nLists
		keys := make([]string, 0, nTotal)
		vals := make([]any, 0, nTotal)
		for i := 0; i < nBools; i++ {
			key := randomString(rng, maxStringSize)
			value := rng.Intn(2) == 1
			keys = append(keys, key)
			vals = append(vals, value)
			line.AddBool(key, value)
		}
		for i := 0; i < nInt64s; i++ {
			key := randomString(rng, maxStringSize)
			value := int64(rng.Intn(maxSafeInteger))
			keys = append(keys, key)
			vals = append(vals, value)
			line.AddInt64(key, value)
		}
		for i := 0; i < nUint64s; i++ {
			key := randomString(rng, maxStringSize)
			value := uint64(rng.Intn(maxSafeInteger))
			keys = append(keys, key)
			vals = append(vals, value)
			line.AddUint64(key, value)
		}
		for i := 0; i < nFloat64s; i++ {
			key := randomString(rng, maxStringSize)
			value := rng.Float64()
			keys = append(keys, key)
			vals = append(vals, value)
			line.AddFloat64(key, value)
		}
		for i := 0; i < nTimes; i++ {
			key := randomString(rng, maxStringSize)
			value := randomTime(rng)
			keys = append(keys, key)
			vals = append(vals, value)
			_ = line.AddTime(key, value)
		}
		for i := 0; i < nStrings; i++ {
			key := randomString(rng, maxStringSize)
			value := randomString(rng, maxStringSize)
			keys = append(keys, key)
			vals = append(vals, value)
			line.AddString(key, value)
		}
		for i := 0; i < nStructs; i++ {
			key := randomString(rng, maxStringSize)
			value := randomPoint(rng)
			keys = append(keys, key)
			vals = append(vals, value)
			_ = line.AddMarshal(key, value)
		}
		for i := 0; i < nRecords; i++ {
			key := randomString(rng, maxStringSize)
			line.StartRecord(key)
			value := toRecord(randomRecord(line, rng, depth+1, maxDepth, maxPairsOfType, maxStringSize))
			line.EndRecord()
			keys = append(keys, key)
			vals = append(vals, value)
		}
		for i := 0; i < nLists; i++ {
			key := randomString(rng, maxStringSize)
			line.StartList(key)
			_, value := randomRecord(line, rng, depth+1, maxDepth, maxPairsOfType, maxStringSize)
			line.EndList()
			keys = append(keys, key)
			vals = append(vals, value)
		}
		return keys, vals
	}

	var buf bytes.Buffer
	enc := goldjson.NewEncoder(&buf)
	line := enc.NewLine()
	record := toRecord(randomRecord(line, rng, 0, maxDepth, maxPairsOfType, maxStringSize))
	_ = line.End()
	expected, _ := json.Marshal(record)
	var createdRecord map[string]any
	err := json.Unmarshal(buf.Bytes(), &createdRecord)
	received, _ := json.Marshal(createdRecord)
	if err != nil {
		t.Fatal(err)
	}
	if i := slicesEqual(expected, received); i != -1 {
		t.Log(string(expected[i:]))
		t.Log(string(received[i:]))
		t.Fatal("mismatched output")
	}
}

func TestDepth(t *testing.T) {
	const maxDepth = 128
	rng := rand.New(rand.NewSource(1))
	var randomRecord func(line *goldjson.LineWriter, rng *rand.Rand, depth, maxDepth int) map[string]any
	randomRecord = func(line *goldjson.LineWriter, rng *rand.Rand, depth, maxDepth int) map[string]any {
		r := make(map[string]any, 1)
		key1 := "a"
		key2 := "b"
		value2 := int64(2)
		if depth == maxDepth {
			value1 := int64(1)
			r[key1] = value1
			r[key2] = value2
			line.AddInt64(key1, value1)
			line.AddInt64(key2, value2)
			return r
		}
		line.StartRecord(key1)
		value1 := randomRecord(line, rng, depth+1, maxDepth)
		line.EndRecord()
		line.AddInt64(key2, value2)
		r[key1] = value1
		r[key2] = value2
		return r
	}

	var buf bytes.Buffer
	enc := goldjson.NewEncoder(&buf)
	line := enc.NewLine()
	record := randomRecord(line, rng, 0, maxDepth)
	_ = line.End()
	expected, _ := json.Marshal(record)
	var createdRecord map[string]any
	err := json.Unmarshal(buf.Bytes(), &createdRecord)
	received, _ := json.Marshal(createdRecord)
	if err != nil {
		t.Fatal(err)
	}
	if i := slicesEqual(expected, received); i != -1 {
		t.Log(string(expected[i:]))
		t.Log(string(received[i:]))
		t.Fatal("mismatched output")
	}
}

func TestListDepth(t *testing.T) {
	const maxDepth = 128
	rng := rand.New(rand.NewSource(1))
	var randomList func(line *goldjson.LineWriter, rng *rand.Rand, depth, maxDepth int) []any
	randomList = func(line *goldjson.LineWriter, rng *rand.Rand, depth, maxDepth int) []any {
		l := make([]any, 0, 2)
		value2 := int64(2)
		if depth == maxDepth {
			value1 := int64(1)
			l = append(l, value1)
			l = append(l, value2)
			line.AddInt64("", value1)
			line.AddInt64("", value2)
			return l
		}
		line.StartList("")
		value1 := randomList(line, rng, depth+1, maxDepth)
		line.EndList()
		line.AddInt64("", value2)
		l = append(l, value1)
		l = append(l, value2)
		return l
	}

	record := map[string]any{}
	var buf bytes.Buffer
	enc := goldjson.NewEncoder(&buf)
	line := enc.NewLine()
	line.StartList("a")
	record["a"] = randomList(line, rng, 0, maxDepth)
	line.EndList()
	_ = line.End()
	expected, _ := json.Marshal(record)
	var createdRecord map[string]any
	err := json.Unmarshal(buf.Bytes(), &createdRecord)
	received, _ := json.Marshal(createdRecord)
	if err != nil {
		t.Fatal(err)
	}
	if i := slicesEqual(expected, received); i != -1 {
		t.Log(string(expected[i:]))
		t.Log(string(received[i:]))
		t.Fatal("mismatched output")
	}
}

func TestPreparedKeys(t *testing.T) {
	fullString := "abcdefhijklmnopqrstuvw0123456789"
	specialCharacters := "\n\r\t\\\""
	rng := rand.New(rand.NewSource(1))
	tests := []struct {
		name  string
		pairs [][2]string
	}{
		{"random 1", [][2]string{{randomASCIIString(rng, 32), randomASCIIString(rng, 32)}}},
		{"random 2", [][2]string{{randomASCIIString(rng, 32), randomASCIIString(rng, 32)}}},
		{"random 3", [][2]string{{randomASCIIString(rng, 32), randomASCIIString(rng, 32)}}},
		{"random 4", [][2]string{{randomASCIIString(rng, 32), randomASCIIString(rng, 32)}}},
		{"substring", [][2]string{{fullString, "abc"}, {fullString[:20], "def"}}},
		{"special characters", [][2]string{{specialCharacters, "abc"}}},
		{"special characters substring", [][2]string{{specialCharacters, "abc"}, {specialCharacters[:2], "def"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bufReceived bytes.Buffer
			var bufExpected bytes.Buffer
			encReceived := goldjson.NewEncoder(&bufReceived)
			encExpected := goldjson.NewEncoder(&bufExpected)
			for _, pair := range tt.pairs {
				encReceived.PrepareKey(pair[0])
			}
			lineReceived := encReceived.NewLine()
			lineExpected := encExpected.NewLine()

			for _, pair := range tt.pairs {
				lineReceived.AddString(pair[0], pair[1])
				lineExpected.AddString(pair[0], pair[1])
			}
			_ = lineReceived.End()
			_ = lineExpected.End()
			received := bufReceived.Bytes()
			expected := bufExpected.Bytes()

			if i := slicesEqual(expected, received); i != -1 {
				t.Log(string(expected[i:]))
				t.Log(string(received[i:]))
				t.Fatal("mismatched output")
			}
		})
	}
}

func TestErrors(t *testing.T) {
	t.Run("invalid time", func(t *testing.T) {
		validTime := baseTime
		invalidTime := time.Date(-1, 06, 12, 20, 42, 15, 152952812, baseZone)
		var buf bytes.Buffer
		enc := goldjson.NewEncoder(&buf)
		expected := `{"valid":"2023-06-12T20:42:15.152952812Z"}` + "\n"

		line := enc.NewLine()
		addErr1 := line.AddTime("valid", validTime)
		addErr2 := line.AddTime("invalid", invalidTime)
		endErr := line.End()
		received := buf.String()

		expectNoError(t, addErr1)
		expectError(t, addErr2)
		expectNoError(t, endErr)
		expectEqual(t, expected, received)
	})

	t.Run("invalid marshal", func(t *testing.T) {
		var buf bytes.Buffer
		enc := goldjson.NewEncoder(&buf)
		expected := `{"valid":{"x":12.34,"y":23.45}}` + "\n"

		line := enc.NewLine()
		addErr1 := line.AddMarshal("valid", Point{12.34, 23.45})
		addErr2 := line.AddMarshal("invalid", ErrorMarshal{})
		endErr := line.End()
		received := buf.String()

		expectNoError(t, addErr1)
		expectError(t, addErr2)
		expectNoError(t, endErr)
		expectEqual(t, expected, received)
	})

	t.Run("cannot write", func(t *testing.T) {
		enc := goldjson.NewEncoder(ErrorWriter{})

		line := enc.NewLine()
		err := line.End()

		expectError(t, err)
	})
}

func Benchmark(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	benches := []struct {
		name        string
		keys        []string
		stringValue string
	}{
		{
			"ascii",
			[]string{
				randomASCIIString(rng, 16),
				randomASCIIString(rng, 16),
				randomASCIIString(rng, 16),
				randomASCIIString(rng, 16),
				randomASCIIString(rng, 16),
				randomASCIIString(rng, 16),
				randomASCIIString(rng, 16),
			},
			randomASCIIString(rng, 256),
		},
		{
			"arbitrary",
			[]string{
				randomString(rng, 16),
				randomString(rng, 16),
				randomString(rng, 16),
				randomString(rng, 16),
				randomString(rng, 16),
				randomString(rng, 16),
				randomString(rng, 16),
			},
			randomString(rng, 256),
		},
	}

	timeValue := randomTime(rng)
	pointValue := randomPoint(rng)

	buf := bytes.NewBuffer(make([]byte, 0, 1024*1024))

	for _, bb := range benches {
		jsonEnc := json.NewEncoder(buf)
		goldEnc := goldjson.NewEncoder(buf)
		goldEncWithKeys := goldjson.NewEncoder(buf)
		for _, key := range bb.keys {
			goldEncWithKeys.PrepareKey(key)
		}

		b.Run(bb.name, func(b *testing.B) {
			b.Run("encoding/json", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					buf.Reset()
					// this comparison is "unfair" on purpose, as the intended
					// use case is encoding more or less dynamic records, and
					// the only way to do that with stdlib JSON is by using
					// dynamically populated maps
					m := map[string]any{}
					m[bb.keys[0]] = bb.stringValue
					m[bb.keys[1]] = uint64(123456789)
					m[bb.keys[2]] = int64(-123456789)
					m[bb.keys[3]] = float64(-123456.789)
					m[bb.keys[4]] = n%2 == 1
					m[bb.keys[5]] = timeValue
					m[bb.keys[6]] = pointValue
					_ = jsonEnc.Encode(m)
				}
			})

			b.Run("goldjson", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					buf.Reset()
					line := goldEnc.NewLine()
					line.AddString(bb.keys[0], bb.stringValue)
					line.AddUint64(bb.keys[1], 123456789)
					line.AddInt64(bb.keys[2], -123456789)
					line.AddFloat64(bb.keys[3], -123456.789)
					line.AddBool(bb.keys[4], n%2 == 1)
					_ = line.AddTime(bb.keys[5], timeValue)
					_ = line.AddMarshal(bb.keys[6], pointValue)
					_ = line.End()
				}
			})

			b.Run("goldjson known keys", func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					buf.Reset()
					line := goldEncWithKeys.NewLine()
					line.AddString(bb.keys[0], bb.stringValue)
					line.AddUint64(bb.keys[1], 123456789)
					line.AddInt64(bb.keys[2], -123456789)
					line.AddFloat64(bb.keys[3], -123456.789)
					line.AddBool(bb.keys[4], n%2 == 1)
					_ = line.AddTime(bb.keys[5], timeValue)
					_ = line.AddMarshal(bb.keys[6], pointValue)
					_ = line.End()
				}
			})
		})
	}
}

func randomASCIIString(rng *rand.Rand, maxStringSize int) string {
	b := make([]byte, 1+rng.Intn(maxStringSize))
	_, _ = rng.Read(b)
	first := byte('a')
	last := byte('z')
	offset := last - first + 1
	for i := range b {
		b[i] = first + (b[i] % offset)
	}
	return string(b)
}

func randomString(rng *rand.Rand, maxStringSize int) string {
	b := make([]byte, 1+rng.Intn(maxStringSize))
	_, _ = rng.Read(b)
	i := 0
	for {
		if i >= len(b) {
			break
		}
		_, s := utf8.DecodeRune(b[i:])
		if s == 0 {
			_, _ = rng.Read(b[i:])
			continue
		}
		i += s
	}
	return string(b)
}

var baseZone = time.FixedZone("night city", 0)
var baseTime = time.Date(2023, 06, 12, 20, 42, 15, 152952812, baseZone)

func randomTime(rng *rand.Rand) time.Time {
	return baseTime.Add(time.Duration(rng.Intn(int(time.Hour) * 24)))
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func randomPoint(rng *rand.Rand) Point {
	return Point{X: rng.Float64(), Y: rng.Float64()}
}

type ErrorMarshal struct{}

func (m ErrorMarshal) MarshalJSON() ([]byte, error) {
	return nil, errors.New("failed")
}

type ErrorWriter struct{}

func (ErrorWriter) Write([]byte) (n int, err error) {
	return 0, errors.New("failed")
}

func slicesEqual[T comparable](a, b []T) int {
	if len(a) > len(b) {
		return len(b)
	}
	if len(b) > len(a) {
		return len(a)
	}
	for i := range a {
		if a[i] != b[i] {
			return i
		}
	}
	return -1
}

func expectNoError(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("expected no error, got %##v", err)
	}
}

func expectError(tb testing.TB, err error) {
	tb.Helper()
	if err == nil {
		tb.Fatalf("expected error, got <nil>")
	}
}

func expectEqual[T comparable](tb testing.TB, expected, received T) {
	tb.Helper()
	if expected != received {
		tb.Fatalf("expected %##v, got %##v", expected, received)
	}
}
