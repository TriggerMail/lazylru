package main

import (
	"time"

	hlru "github.com/hashicorp/golang-lru"
)

type hashicorpCache interface {
	Get(key interface{}) (value interface{}, ok bool)
	Add(key, value interface{})
	Purge()
}

// ignoreEvict is a type alias to make the Add method of Cache like the others
type ignoreEvict struct {
	*hlru.Cache
}

func (c *ignoreEvict) Get(key interface{}) (value interface{}, ok bool) {
	return c.Cache.Get(key)
}

func (c *ignoreEvict) Add(key, value interface{}) {
	c.Cache.Add(key, value)
}

func (c *ignoreEvict) Purge() {
	c.Cache.Purge()
}

// HashicorpWrapper wraps hashicorp/golang-lru.Cache in an interface for testing
type HashicorpWrapper[K comparable, V any] struct {
	cache hashicorpCache
}

// Get looks an item up in the cache. The boolean indicates if the key was found
func (c *HashicorpWrapper[K, V]) Get(key K) (V, bool) {
	var retval V
	v, ok := c.cache.Get(key)
	if ok {
		retval, ok = v.(V)
	}
	return retval, ok
}

// Set adds or updates a value in the cache
func (c *HashicorpWrapper[K, V]) Set(key string, value string) {
	c.cache.Add(key, value)
}

// Close removes everything from the cache
func (c *HashicorpWrapper[K, V]) Close() {
	c.cache.Purge()
}

// NewHashicorpWrapper initializes a new cache with capacity of `size`
func NewHashicorpWrapper[K comparable, V any](size int) *HashicorpWrapper[K, V] {
	retval, err := hlru.New(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpWrapper[K, V]{&ignoreEvict{retval}}
}

// NewHashicorpARCWrapper initializes a new cache with capacity of `size`
func NewHashicorpARCWrapper[K comparable, V any](size int) *HashicorpWrapper[K, V] {
	retval, err := hlru.NewARC(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpWrapper[K, V]{retval}
}

// NewHashicorp2QWrapper initializes a new cache with capacity of `size`
func NewHashicorp2QWrapper[K comparable, V any](size int) *HashicorpWrapper[K, V] {
	retval, err := hlru.New2Q(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpWrapper[K, V]{retval}
}

// HashicorpWrapperExp is a wrapper around the hashicorp.cache that stores items
// in an envelope that enforces a time-to-live constraint
type HashicorpWrapperExp[K comparable, V any] struct {
	cache *hlru.Cache
	ttl   time.Duration
}

// NewHashicorpWrapperExp initializes a new cache with capacity of `size`
func NewHashicorpWrapperExp[K comparable, V any](size int, ttl time.Duration) *HashicorpWrapperExp[K, V] {
	retval, err := hlru.New(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpWrapperExp[K, V]{retval, ttl}
}

// Get looks an item up in the cache. The boolean indicates if the key was found
func (c *HashicorpWrapperExp[K, V]) Get(key K) (V, bool) {
	var zeroval V
	ret, ok := c.cache.Get(key)
	if !ok {
		return zeroval, false
	}
	item, ok := ret.(mapCacheElement[V])
	if !ok {
		return zeroval, false
	}
	if item.expiration.Before(time.Now()) {
		return zeroval, ok
	}

	return item.value, ok
}

// Set adds or updates a value in the cache
func (c *HashicorpWrapperExp[K, V]) Set(key K, value V) {
	c.cache.Add(key, mapCacheElement[V]{value: value, expiration: time.Now().Add(c.ttl)})
}

// Close removes everything from the cache
func (c *HashicorpWrapperExp[K, V]) Close() {
	c.cache.Purge()
}
