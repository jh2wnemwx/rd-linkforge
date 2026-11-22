// Package topk keeps the N best items of a stream in a bounded heap.
package topk

import (
	"container/heap"
	"errors"
	"slices"
	"strings"
)

// ErrInvalidN is returned by New for an n of zero or less.
var ErrInvalidN = errors.New("topk: n must be positive")

// Item is a key with its count.
type Item struct {
	Key   string
	Count int64
}

// better reports whether a ranks above b: a higher Count wins; with equal
// counts the smaller Key wins.
func better(a, b Item) bool {
	if a.Count != b.Count {
		return a.Count > b.Count
	}
	return a.Key < b.Key
}

// minHeap implements heap.Interface with the worst kept item at the root.
type minHeap []Item

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return better(h[j], h[i]) }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(Item)) }
func (h *minHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}

// TopK keeps the n best items offered to it.
type TopK struct {
	n    int
	heap minHeap
}

// New returns ErrInvalidN when n is zero or negative.
func New(n int) (*TopK, error) {
	if n <= 0 {
		return nil, ErrInvalidN
	}
	return &TopK{n: n, heap: make(minHeap, 0, n)}, nil
}

// Add offers an item. Items with equal keys are not merged. Add is O(log n).
func (t *TopK) Add(key string, count int64) {
	it := Item{Key: key, Count: count}
	if t.heap.Len() < t.n {
		heap.Push(&t.heap, it)
		return
	}
	if better(it, t.heap[0]) {
		heap.Pop(&t.heap)
		heap.Push(&t.heap, it)
	}
}

// Result returns the kept items, best first. It does not change t.
func (t *TopK) Result() []Item {
	items := slices.Clone([]Item(t.heap))
	slices.SortFunc(items, func(a, b Item) int {
		if a.Count != b.Count {
			if a.Count > b.Count {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Key, b.Key)
	})
	return items
}
