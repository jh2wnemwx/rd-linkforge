package cache

import (
	"container/list"
	"errors"
	"sync"
)

var ErrInvalidCapacity = errors.New("cache: capacity must be positive")

type LRU[K comparable, V any] struct {
	mu sync.Mutex
	capacity int
	order *list.List
	items map[K]*list.Element
}

type entry[K comparable, V any] struct {
	key K
	value V
}

var _ Cache[string, int] = (*LRU[string, int])(nil)

// NewLRU returns ErrInvalidCapacity when capacity is zero or negative.
func NewLRU[K comparable, V any](capacity int) (*LRU[K, V], error) {
	if capacity < 0 {
		return nil, ErrInvalidCapacity
	}

	return &LRU[K, V] {
		capacity: capacity,
		order: list.New(),
		items: make(map[K]*list.Element),
	}, nil
}	

func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false 
	}

	c.order.MoveToFront(e)
	return e.Value.(*entry[K, V]).value, true
}

func (c *LRU[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		e.Value.(*entry[K, V]).value = value
		c.order.MoveToFront(e)
		return
	}

	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*entry[K, V]).key)
	}

	c.items[key] = c.order.PushFront(&entry[K, V]{key: key, value: value})
}

func (c *LRU[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		c.order.Remove(e)
		delete(c.items, key)
	}
}

func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}

// Keys returns the keys from the most recently used to the least recently used.
// It does not change the order.
func (c *LRU[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]K, 0, c.order.Len())
	for e := c.order.Front(); e != nil; e.Next() {
		keys = append(keys, e.Value.(*entry[K, V]).key)
	}

	return keys
}