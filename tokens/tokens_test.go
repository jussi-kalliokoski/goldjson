package tokens_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jussi-kalliokoski/goldjson/tokens"
)

func TestAppendInt64(t *testing.T) {
	tests := []struct {
		val      int64
		expected string
	}{
		{591824, "591824"},
		{1e6, "1000000"},
		{-2e6, "-2000000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			received := string(tokens.AppendInt64(nil, tt.val))

			expectEqual(t, tt.expected, received)
		})
	}
}

func TestAppendUint64(t *testing.T) {
	tests := []struct {
		val      uint64
		expected string
	}{
		{591824, "591824"},
		{1e6, "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			received := string(tokens.AppendUint64(nil, tt.val))

			expectEqual(t, tt.expected, received)
		})
	}
}

func TestAppendBool(t *testing.T) {
	tests := []struct {
		val      bool
		expected string
	}{
		{false, "false"},
		{true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			received := string(tokens.AppendBool(nil, tt.val))

			expectEqual(t, tt.expected, received)
		})
	}
}

func TestAppendFloat64(t *testing.T) {
	t.Run("special", func(t *testing.T) {
		z := float64(0)
		tests := []struct {
			name     string
			val      float64
			expected string
		}{
			{"positive infinity", 1 / z, "+Inf"},
			{"negative infinity", -1 / z, "-Inf"},
			{"NaN", 0 / z, "NaN"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expected := fmt.Sprintf("%q", tt.expected)

				received := string(tokens.AppendFloat64(nil, tt.val))

				expectEqual(t, expected, received)
			})
		}
	})

	t.Run("normal", func(t *testing.T) {
		tests := []struct {
			val      float64
			expected string
		}{
			{0.591824, "0.591824"},
			{1e-09, "1e-9"},
			{1e-12, "1e-12"},
			{1e-6, "0.000001"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				received := string(tokens.AppendFloat64(nil, tt.val))

				expectEqual(t, tt.expected, received)
			})
		}
	})
}

func TestAppendString(t *testing.T) {
	t.Run("special", func(t *testing.T) {
		tests := []struct {
			name     string
			key      []byte
			expected string
		}{
			{"first non-utf8", []byte{255, 0}, "\"\\ufffd\\u0000\""},
			{"second non-utf8", []byte{'a', 255, 0}, "\"a\\ufffd\\u0000\""},
			{"first line separator", []byte("\u2028"), "\"\\u2028\""},
			{"second line separator", []byte("a\u2028"), "\"a\\u2028\""},
			{"first paragraph separator", []byte("\u2029"), "\"\\u2029\""},
			{"second paragraph separator", []byte("a\u2029"), "\"a\\u2029\""},
			{"first bullet", []byte("\u2022"), "\"\u2022\""},
			{"second bullet", []byte("a\u2022"), "\"a\u2022\""},
			{"first newline", []byte("\n"), "\"\\n\""},
			{"second newline", []byte("a\n"), "\"a\\n\""},
			{"first carriage return", []byte("\r"), "\"\\r\""},
			{"second carriage return", []byte("a\r"), "\"a\\r\""},
			{"first tab", []byte("\t"), "\"\\t\""},
			{"second tab", []byte("a\t"), "\"a\\t\""},
			{"first backslash", []byte("\\"), "\"\\\\\""},
			{"second backslash", []byte("a\\"), "\"a\\\\\""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				key := string(tt.key)

				received := string(tokens.AppendString(nil, key))

				expectEqual(t, tt.expected, received)
			})
		}
	})

	t.Run("normal", func(t *testing.T) {
		tests := []struct {
			name     string
			key      string
			expected string
		}{
			{"abc", "abc", `"abc"`},
			{"123", "123", `"123"`},
			{"<>", "<>", `"<>"`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				received := string(tokens.AppendString(nil, tt.key))

				expectEqual(t, tt.expected, received)
			})
		}
	})
}

func TestAppendMarshal(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name     string
			val      any
			expected string
		}{
			{"Point", Point{1.23, -0.5}, `{"x":1.23,"y":-0.5}`},
			{"CustomMarshal", CustomMarshal{"html", "<>"}, `{"html":"<>"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b, err := tokens.AppendMarshal(nil, tt.val)
				received := string(b)

				expectNoError(t, err)
				expectEqual(t, tt.expected, received)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name string
			val  any
		}{
			{"ErrorMarshal", ErrorMarshal{}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expected := []byte("abc")
				received, err := tokens.AppendMarshal(expected, tt.val)

				expectError(t, err)
				expectEqual(t, string(expected), string(received))
			})
		}
	})
}

func TestAppendTime(t *testing.T) {
	zone := time.FixedZone("night city", 0)

	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name     string
			val      time.Time
			expected string
		}{
			{"future", time.Date(2077, 06, 12, 20, 42, 15, 152952812, zone), `"2077-06-12T20:42:15.152952812Z"`},
			{"past", time.Date(1234, 1, 2, 3, 4, 5, 6, zone), `"1234-01-02T03:04:05.000000006Z"`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b, err := tokens.AppendTime(nil, tt.val)
				received := string(b)

				expectNoError(t, err)
				expectEqual(t, tt.expected, received)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name string
			val  time.Time
		}{
			{"year too far in the past", time.Date(-1, 06, 12, 20, 42, 15, 152952, zone)},
			{"year too far in the future", time.Date(10000, 06, 12, 20, 42, 15, 152952, zone)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expected := []byte("abc")
				received, err := tokens.AppendTime(expected, tt.val)

				expectError(t, err)
				expectEqual(t, string(expected), string(received))
			})
		}
	})
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type CustomMarshal struct {
	K string
	V string
}

func (m CustomMarshal) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{%q:%q}", m.K, m.V)), nil
}

type ErrorMarshal struct{}

func (m ErrorMarshal) MarshalJSON() ([]byte, error) {
	return nil, errors.New("failed")
}

func expectEqual[T comparable](tb testing.TB, expected, received T) {
	tb.Helper()
	if expected != received {
		tb.Fatalf("expected %##v, got %##v", expected, received)
	}
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
