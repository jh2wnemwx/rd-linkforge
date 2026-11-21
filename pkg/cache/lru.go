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

func (c *LRU[K, V]) Put(key K, value V)
func (c *LRU[K, V]) Delete(key K)
func (c *LRU[K, V]) Len() int

// Keys returns the keys from the most recently used to the least recently used.
// It does not change the order.
func (c *LRU[K, V]) Keys() []K