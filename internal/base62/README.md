# Package base62 

## Implements the following public functions

Encode renders a number with no padding.
```
func Encode(v uint64) string
```
EncodeWidth does the same but left-pads with zeroes to at least width characters.  
The value itself is never truncated.
```
func EncodeWidth(v uint64, width int) string
```
Decode parses a code back into a number.
```
func Decode(s string) (uint64, error)
```

## This package adds the following new errors
```
var (
    ErrEmpty             = errors.New("base62: empty code")
    ErrInvalidCharacter  = errors.New("base62: invalid character")
    ErrOverflow          = errors.New("base62: code is too long")
)
```

## Implementation notes
Apparently, the maximum base62 value that fits in uint64 is `LygHa16AHYF`, so we needed to introduce an additional check in Decode to handle potential type capacity overflow.  
Tests have been updated accordingly to cover this case.