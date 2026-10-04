package base62

import (
	"errors"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrEmpty            = errors.New("base62: empty code")
	ErrInvalidCharacter = errors.New("base62: invalid character")
	ErrOverflow         = errors.New("base62: code is too long")
)

// Encode renders a number with no padding.
func Encode(v uint64) string {
	return ""
}

// EncodeWidth does the same but left-pads with zeroes to at least width
// characters. The value itself is never truncated.
func EncodeWidth(v uint64, width int) string {
	return ""
}

// Decode parses a code back into a number.
func Decode(s string) (uint64, error) {
	return 0, nil
}

func decodeDigit(c byte) (byte, bool) {
	return 0, false
}
