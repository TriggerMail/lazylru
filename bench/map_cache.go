package main

import (
	"sync"
	"time"
)

type mapCacheElement[V any] struct {
	expiration time.Time
	value      V
}

// MapCache is a simple, map-based implementation of a cache. The replacement
// "strategy" is random. All operations take an exclusive lock. It is meant to
// be the control in our experiment.
type MapCache[K comparable, V any] struct {
	mut     sync.Mutex
	data    map[K]mapCacheElement[V]
	ttl     time.Duration
	maxSize int
}

// NewMapCache creates a new cache with a given size and item time-to-live
func NewMapCache[K comparable, V any](maxSize int, ttl time.Duration) *MapCache[K, V] {
	return &MapCache[K, V]{
		data:    map[K]mapCacheElement[V]{},
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves an item from the cache
func (mc *MapCache[K, V]) Get(key K) (V, bool) {
	mc.mut.Lock()
	ret, ok := mc.data[key]
	if ok && ret.expiration.Before(time.Now()) {
		delete(mc.data, key)
		ok = false
	}
	mc.mut.Unlock()
	if ok {
		return ret.value, ok
	}
	var zeroval V
	return zeroval, false
}

// Set adds an item to the cache
func (mc *MapCache[K, V]) Set(key K, value V) {
	mc.mut.Lock()
	// are we about to run out of space?
	if len(mc.data) == mc.maxSize {
		_, ok := mc.data[key]
		// if we are not replacing an existing element, delete an element arbitrarily
		if !ok {
			// is it worth the trouble to try to find expired elements?
			// probably not
			var k K
			for k = range mc.data {
				break
			}
			delete(mc.data, k)
		}
	}
	mc.data[key] = mapCacheElement[V]{
		value:      value,
		expiration: time.Now().Add(mc.ttl),
	}
	mc.mut.Unlock()
}

// Close doesn't do anything in this implementation
func (mc *MapCache[K, V]) Close() {}
