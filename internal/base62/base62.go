package base62

import (
	"errors"
	"strings"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrEmpty            = errors.New("base62: empty code")
	ErrInvalidCharacter = errors.New("base62: invalid character")
	ErrOverflow         = errors.New("base62: code is too long")
)

// Encode renders a number with no padding.
func Encode(v uint64) string {
	if v == 0 {
		return "0"
	}

	// Max uint64 fits in 11 base62 characters
	var s [11]byte

	i := len(s)
	for v > 0 {
		r := v % 62
		v /= 62
		i--
		s[i] = alphabet[r]
	}
	return string(s[i:])
}

// EncodeWidth does the same but left-pads with zeroes to at least width
// characters. The value itself is never truncated.
func EncodeWidth(v uint64, width int) string {
	enc := Encode(v)
	if len(enc) > width {
		return enc
	}

	return strings.Repeat("0", width-len(enc)) + enc
}

// Decode parses a code back into a number.
func Decode(s string) (uint64, error) {
	if s == "" {
		return 0, ErrEmpty
	}

	if len(s) > 11 {
		return 0, ErrOverflow
	}

	var v uint64

	for i := range len(s) {
		d, ok := unbase62(s[i])
		if !ok {
			return 0, ErrInvalidCharacter
		}

		// Check if v * 62 + d can fit in uint64
		if v > (^uint64(0)-uint64(d))/62 {
			return 0, ErrOverflow
		}
		v = v*62 + uint64(d)
	}

	return v, nil
}

func unbase62(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'A' <= c && c <= 'Z':
		return c - 'A' + 10, true
	case 'a' <= c && c <= 'z':
		return c - 'a' + 36, true

	default:
		return 0, false
	}
}
