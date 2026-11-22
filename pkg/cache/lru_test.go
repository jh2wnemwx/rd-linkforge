package cache

import (
	"errors"
	"slices"
	"sync"
	"testing"
)

func newLRU(t *testing.T, capacity int) *LRU[string, int] {
	t.Helper()
	c, err := NewLRU[string, int](capacity)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func wantKeys(t *testing.T, c *LRU[string, int], want ...string) {
	t.Helper()
	if got := c.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
	if c.Len() != len(want) {
		t.Errorf("Len() = %d, want %d", c.Len(), len(want))
	}
}

func TestNewLRUInvalidCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		c, err := NewLRU[string, int](capacity)
		if !errors.Is(err, ErrInvalidCapacity) || c != nil {
			t.Errorf("NewLRU(%d) = %v, %v; want nil, ErrInvalidCapacity", capacity, c, err)
		}
	}
}

func TestLRUGet(t *testing.T) {
	c := newLRU(t, 3)
	if v, ok := c.Get("missing"); ok || v != 0 {
		t.Errorf("Get on empty cache = %d, %v; want 0, false", v, ok)
	}

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)
	wantKeys(t, c, "c", "b", "a")

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Errorf("Get(a) = %d, %v; want 1, true", v, ok)
	}
	wantKeys(t, c, "a", "c", "b") // a hit moves the entry to the front

	if _, ok := c.Get("x"); ok {
		t.Error("Get(x) hit, want a miss")
	}
	wantKeys(t, c, "a", "c", "b") // a miss changes nothing
}

func TestLRUPutExistingKey(t *testing.T) {
	c := newLRU(t, 3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("a", 10)

	wantKeys(t, c, "a", "b")
	if v, _ := c.Get("a"); v != 10 {
		t.Errorf("Get(a) = %d, want 10", v)
	}
}

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	c := newLRU(t, 2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Get("a")    // b is now the least recently used
	c.Put("c", 3) // evicts b

	wantKeys(t, c, "c", "a")
	if _, ok := c.Get("b"); ok {
		t.Error("b was not evicted")
	}

	c.Put("a", 4) // replacing a key in a full cache evicts nothing
	wantKeys(t, c, "a", "c")
}

func TestLRUCapacityOne(t *testing.T) {
	c := newLRU(t, 1)
	c.Put("a", 1)
	c.Put("b", 2)
	wantKeys(t, c, "b")
}

func TestLRUDelete(t *testing.T) {
	c := newLRU(t, 3)
	c.Put("a", 1)
	c.Put("b", 2)

	c.Delete("missing") // does nothing
	wantKeys(t, c, "b", "a")

	c.Delete("b")
	wantKeys(t, c, "a")
	if _, ok := c.Get("b"); ok {
		t.Error("Get(b) hit after Delete")
	}

	// The freed slot is reused without evicting a.
	c.Put("c", 3)
	c.Put("d", 4)
	wantKeys(t, c, "d", "c", "a")
}

func TestLRUKeysDoesNotChangeOrder(t *testing.T) {
	c := newLRU(t, 3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Keys()
	c.Keys()
	c.Put("c", 3)
	c.Put("d", 4) // must evict a, the least recently used
	wantKeys(t, c, "d", "c", "b")
}

func TestLRUConcurrentUse(t *testing.T) {
	c := newLRU(t, 8)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := string(rune('a' + g))
			for i := range 1000 {
				c.Put(key, i)
				c.Get(key)
				c.Keys()
				if i%10 == 0 {
					c.Delete(key)
				}
			}
		}()
	}
	wg.Wait()
	if c.Len() > 8 {
		t.Errorf("Len() = %d, over capacity", c.Len())
	}
}
