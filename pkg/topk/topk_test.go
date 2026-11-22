package topk

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

func newTopK(t *testing.T, n int) *TopK {
	t.Helper()
	tk, err := New(n)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func TestNewInvalidN(t *testing.T) {
	for _, n := range []int{0, -3} {
		tk, err := New(n)
		if !errors.Is(err, ErrInvalidN) || tk != nil {
			t.Errorf("New(%d) = %v, %v; want nil, ErrInvalidN", n, tk, err)
		}
	}
}

func TestExample(t *testing.T) {
	tk := newTopK(t, 3)
	for i, c := range []int64{5, 1, 9, 3, 9, 7, 2} {
		tk.Add(fmt.Sprintf("k%d", i), c)
	}
	want := []Item{{"k2", 9}, {"k4", 9}, {"k5", 7}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestFewerThanN(t *testing.T) {
	tk := newTopK(t, 5)
	if got := tk.Result(); len(got) != 0 {
		t.Errorf("empty Result() = %v", got)
	}
	tk.Add("a", 1)
	tk.Add("b", 3)
	want := []Item{{"b", 3}, {"a", 1}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestTiesBreakBySmallerKey(t *testing.T) {
	tk := newTopK(t, 2)
	tk.Add("c", 5)
	tk.Add("b", 5)
	tk.Add("d", 5) // equal to the worst kept (c) and not better: rejected
	tk.Add("a", 5) // better than c: replaces it
	want := []Item{{"a", 5}, {"b", 5}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestEqualItemIsNotBetter(t *testing.T) {
	tk := newTopK(t, 1)
	tk.Add("a", 5)
	tk.Add("a", 5) // same key and count: not strictly better
	tk.Add("b", 4)
	want := []Item{{"a", 5}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestEqualKeysAreNotMerged(t *testing.T) {
	tk := newTopK(t, 3)
	tk.Add("a", 1)
	tk.Add("a", 2)
	want := []Item{{"a", 2}, {"a", 1}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestNegativeCounts(t *testing.T) {
	tk := newTopK(t, 2)
	for i, c := range []int64{-5, -1, -9, 0} {
		tk.Add(fmt.Sprintf("k%d", i), c)
	}
	want := []Item{{"k3", 0}, {"k1", -1}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestResultDoesNotChangeTopK(t *testing.T) {
	tk := newTopK(t, 3)
	for i, c := range []int64{4, 8, 6} {
		tk.Add(fmt.Sprintf("k%d", i), c)
	}
	first := tk.Result()
	first[0] = Item{"mutated", 100}

	if got := tk.Result(); got[0] != (Item{"k1", 8}) {
		t.Errorf("Result() after mutating a previous result = %v", got)
	}

	// The heap still works: k0 (4) is the worst and gets replaced.
	tk.Add("k3", 5)
	want := []Item{{"k1", 8}, {"k2", 6}, {"k3", 5}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}

func TestManyItems(t *testing.T) {
	tk := newTopK(t, 5)
	for i := range 1000 {
		// Counts are a permutation of 0..999, so the best five are 995..999.
		c := int64((i * 7919) % 1000)
		tk.Add(fmt.Sprintf("k%03d", c), c)
	}
	want := []Item{{"k999", 999}, {"k998", 998}, {"k997", 997}, {"k996", 996}, {"k995", 995}}
	if got := tk.Result(); !slices.Equal(got, want) {
		t.Errorf("Result() = %v, want %v", got, want)
	}
}
