package shortid

import "testing"

// The skeleton stubs must keep doing nothing: no code, no error.
func TestNoopGenerator(t *testing.T) {
	var g Generator = NoopGenerator{}
	code, err := g.Next()
	if code != "" || err != nil {
		t.Errorf("Next() = %q, %v; want \"\", nil", code, err)
	}
}

func TestNoopCodec(t *testing.T) {
	var c Codec = NoopCodec{}
	if got := c.Encode(42); got != "" {
		t.Errorf("Encode(42) = %q, want \"\"", got)
	}
	n, err := c.Decode("abc")
	if n != 0 || err != nil {
		t.Errorf("Decode(\"abc\") = %d, %v; want 0, nil", n, err)
	}
}
