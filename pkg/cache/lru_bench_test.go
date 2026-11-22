package cache

import (
	"strconv"
	"testing"
)

// BenchmarkLRUGetHit measures Get on keys that are all in the cache, so every
// call is a hit that also moves the entry to the front.
func BenchmarkLRUGetHit(b *testing.B) {
	const size = 1000
	c, err := NewLRU[string, int](size)
	if err != nil {
		b.Fatal(err)
	}
	keys := make([]string, size)
	for i := range keys {
		keys[i] = "code" + strconv.Itoa(i)
		c.Put(keys[i], i)
	}

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		if _, ok := c.Get(keys[i%size]); !ok {
			b.Fatalf("miss on %q", keys[i%size])
		}
		i++
	}
}
