package trie

import (
	"strconv"
	"testing"
)

// benchWords returns n distinct 6-character base36 words. They are spread over
// the keyspace by a fixed multiplier, so the set is the same on every run.
func benchWords(n int) []string {
	const space = 36 * 36 * 36 * 36 * 36 * 36 // 36^6 six-character words
	words := make([]string, n)
	for i := range words {
		v := (uint64(i) * 2654435761) % space // odd multiplier: distinct for i < space
		w := strconv.FormatUint(v, 36)
		for len(w) < 6 {
			w = "0" + w
		}
		words[i] = w
	}
	return words
}

// BenchmarkWithPrefix looks up a one-character prefix in a trie of 10 000
// words: once capped at 10 results (autocomplete) and once for all matches.
func BenchmarkWithPrefix(b *testing.B) {
	tr := New()
	for _, w := range benchWords(10_000) {
		tr.Insert(w)
	}
	const prefix = "a"

	for _, bc := range []struct {
		name  string
		limit int
	}{
		{"limit=10", 10},
		{"all", 0},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			var got []string
			for b.Loop() {
				got = tr.WithPrefix(prefix, bc.limit)
			}
			if len(got) == 0 {
				b.Fatalf("no words start with %q", prefix)
			}
			b.ReportMetric(float64(len(got)), "words/op")
		})
	}
}
