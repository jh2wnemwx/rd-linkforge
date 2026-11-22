package shortid

import (
	"fmt"
	"io"
)

const (
	codeWidth   = 6
	randomBits  = 19
	counterBits = 16
	alphabet    = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// CodeGenerator issues 6-character base62 codes from a 35-bit value: 19 random
// high bits and a 16-bit counter in the low bits. It is not safe for
// concurrent use.
type CodeGenerator struct {
	random  io.Reader
	counter uint16 // wraps to 0 after 2^16 - 1
}

var _ Generator = (*CodeGenerator)(nil)

// NewCodeGenerator takes the source of random bytes. The application will pass
// crypto/rand.Reader; tests pass a fixed reader.
func NewCodeGenerator(random io.Reader) *CodeGenerator {
	return &CodeGenerator{random: random}
}

// Next returns the next code. If reading the random bytes fails, it returns
// "" and an error wrapping the reader's error, and the counter is unchanged.
func (g *CodeGenerator) Next() (string, error) {
	var b [3]byte
	if _, err := io.ReadFull(g.random, b[:]); err != nil {
		return "", fmt.Errorf("shortid: read random bytes: %w", err)
	}

	r := (uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])) & (1<<randomBits - 1)
	v := r<<counterBits | uint64(g.counter)
	g.counter++ // uint16 wraps to 0 after 65535
	return encode(v), nil
}

// encode writes n in base62 over exactly codeWidth characters, padded on the
// left with '0'. n must be below 62^codeWidth, which every 35-bit value is.
func encode(n uint64) string {
	var buf [codeWidth]byte
	for i := codeWidth - 1; i >= 0; i-- {
		buf[i] = alphabet[n%62]
		n /= 62
	}
	return string(buf[:])
}
