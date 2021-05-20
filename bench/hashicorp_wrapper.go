package main

import (
	"time"

	hlru "github.com/hashicorp/golang-lru"
)

type HashicorpWrapper struct {
	cache *hlru.Cache
}

func NewHashicorpWrapper(size int) *HashicorpWrapper {
	retval, err := hlru.New(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpWrapper{retval}
}

func (c *HashicorpWrapper) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *HashicorpWrapper) Set(key string, value interface{}) {
	c.cache.Add(key, value)
}

func (c *HashicorpWrapper) Close() {
	c.cache.Purge()
}

type HashicorpARCWrapper struct {
	cache *hlru.ARCCache
}

func NewHashicorpARCWrapper(size int) *HashicorpARCWrapper {
	retval, err := hlru.NewARC(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpARCWrapper{retval}
}

func (c *HashicorpARCWrapper) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *HashicorpARCWrapper) Set(key string, value interface{}) {
	c.cache.Add(key, value)
}

func (c *HashicorpARCWrapper) Close() {
	c.cache.Purge()
}

type Hashicorp2QWrapper struct {
	cache *hlru.TwoQueueCache
}

func NewHashicorp2QWrapper(size int) *Hashicorp2QWrapper {
	retval, err := hlru.New2Q(size)
	if err != nil {
		panic(err)
	}
	return &Hashicorp2QWrapper{retval}
}

func (c *Hashicorp2QWrapper) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

func (c *Hashicorp2QWrapper) Set(key string, value interface{}) {
	c.cache.Add(key, value)
}

func (c *Hashicorp2QWrapper) Close() {
	c.cache.Purge()
}

type HashicorpWrapperExp struct {
	cache *hlru.Cache
	ttl   time.Duration
}

func NewHashicorpWrapperExp(size int, ttl time.Duration) *HashicorpWrapperExp {
	retval, err := hlru.New(size)
	if err != nil {
		panic(err)
	}
	return &HashicorpWrapperExp{retval, ttl}
}

func (c *HashicorpWrapperExp) Get(key string) (interface{}, bool) {
	ret, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	item, _ := ret.(mapCacheElement)
	if item.expiration.Before(time.Now()) {
		return nil, ok
	}

	return ret, ok
}

func (c *HashicorpWrapperExp) Set(key string, value interface{}) {
	c.cache.Add(key, mapCacheElement{value: value, expiration: time.Now().Add(c.ttl)})
}

func (c *HashicorpWrapperExp) Close() {
	c.cache.Purge()
}
