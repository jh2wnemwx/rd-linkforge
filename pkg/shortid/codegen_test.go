package shortid

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// repeatReader returns the same bytes on every read, forever.
type repeatReader struct{ b []byte }

func (r repeatReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b[i%len(r.b)]
	}
	return len(p), nil
}

// failAfter serves n good reads of zeros, then fails with err.
type failAfter struct {
	n   int
	err error
}

func (r *failAfter) Read(p []byte) (int, error) {
	if r.n == 0 {
		return 0, r.err
	}
	r.n--
	clear(p)
	return len(p), nil
}

func next(t *testing.T, g *CodeGenerator) string {
	t.Helper()
	code, err := g.Next()
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func TestReferenceValues(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  []string
	}{
		{"00 00 01", []byte{0x00, 0x00, 0x01}, []string{"000H32", "000H33", "000H34"}},
		{"FF FF FF", []byte{0xFF, 0xFF, 0xFF}, []string{"bVJgVU"}},
		{"00 00 00", []byte{0x00, 0x00, 0x00}, []string{"000000", "000001"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewCodeGenerator(repeatReader{tt.bytes})
			for i, want := range tt.want {
				if got := next(t, g); got != want {
					t.Errorf("call %d: got %q, want %q", i+1, got, want)
				}
			}
		})
	}
}

func TestReadsExactlyThreeBytesFirstMostSignificant(t *testing.T) {
	// The high bits of the first byte are dropped (only 19 bits are kept);
	// the byte after the first three belongs to the second call.
	src := bytes.NewReader([]byte{0xF8, 0x00, 0x01, 0x00, 0x00, 0x00})
	g := NewCodeGenerator(src)
	if got := next(t, g); got != "000H32" {
		t.Errorf("first code = %q, want 000H32", got)
	}
	if got := next(t, g); got != "000001" {
		t.Errorf("second code = %q, want 000001", got)
	}
}

func TestCounterWraps(t *testing.T) {
	g := NewCodeGenerator(repeatReader{[]byte{0, 0, 0}})
	g.counter = 1<<16 - 1
	if got := next(t, g); got != encode(1<<16-1) {
		t.Errorf("got %q, want %q", got, encode(1<<16-1))
	}
	if got := next(t, g); got != "000000" {
		t.Errorf("after 2^16-1 got %q, want 000000", got)
	}
}

func TestDistinctForManyCalls(t *testing.T) {
	g := NewCodeGenerator(repeatReader{[]byte{0x12, 0x34, 0x56}})
	seen := make(map[string]bool, 1<<16)
	for range 1 << 16 {
		code := next(t, g)
		if len(code) != 6 {
			t.Fatalf("code %q is not 6 characters", code)
		}
		if seen[code] {
			t.Fatalf("code %q issued twice", code)
		}
		seen[code] = true
	}
}

func TestReadErrorKeepsCounter(t *testing.T) {
	boom := errors.New("entropy exhausted")
	src := &failAfter{n: 1, err: boom}
	g := NewCodeGenerator(src)

	if got := next(t, g); got != "000000" {
		t.Fatalf("first code = %q", got)
	}

	code, err := g.Next()
	if code != "" || !errors.Is(err, boom) {
		t.Errorf("Next() = %q, %v; want \"\" and an error wrapping %v", code, err, boom)
	}

	// The failed call did not advance the counter.
	src.n = 1
	if got := next(t, g); got != "000001" {
		t.Errorf("code after the failure = %q, want 000001", got)
	}
}

func TestShortRead(t *testing.T) {
	g := NewCodeGenerator(strings.NewReader("ab")) // only 2 of the 3 bytes
	code, err := g.Next()
	if code != "" || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("Next() = %q, %v; want \"\" and io.ErrUnexpectedEOF", code, err)
	}
}

func TestEncode(t *testing.T) {
	tests := map[uint64]string{
		0:            "000000",
		61:           "00000z",
		62:           "000010",
		1<<35 - 1:    "bVJxYV", // the largest 35-bit value
		56800235583:  "zzzzzz", // 62^6 - 1
		1<<16 | 3844: "000I32",
	}
	for n, want := range tests {
		if got := encode(n); got != want {
			t.Errorf("encode(%d) = %q, want %q", n, got, want)
		}
	}
}
