package main

import (
	"sync"
	"time"
)

type mapCacheElement struct {
	expiration time.Time
	value      interface{}
}

type MapCache struct {
	mut     sync.Mutex
	data    map[string]mapCacheElement
	ttl     time.Duration
	maxSize int
}

func NewMapCache(maxSize int, ttl time.Duration) *MapCache {
	return &MapCache{
		data:    map[string]mapCacheElement{},
		ttl:     ttl,
		maxSize: maxSize,
	}
}

func (mc *MapCache) Get(key string) (interface{}, bool) {
	mc.mut.Lock()
	ret, ok := mc.data[key]
	if ok && ret.expiration.Before(time.Now()) {
		delete(mc.data, key)
		ok = false
	}
	mc.mut.Unlock()
	if ok {
		return ret, ok
	}
	return nil, false
}

func (mc *MapCache) Set(key string, value interface{}) {
	mc.mut.Lock()
	// are we about to run out of space?
	if len(mc.data) == mc.maxSize {
		_, ok := mc.data[key]
		// if we are not replacing an existing element, delete an element arbitrarily
		if !ok {
			// is it worth the trouble to try to find expired elements?
			// probably not
			var k string
			for k = range mc.data {
				break
			}
			delete(mc.data, k)
		}
	}
	mc.data[key] = mapCacheElement{
		value:      value,
		expiration: time.Now().Add(mc.ttl),
	}
	mc.mut.Unlock()
}

func (mc *MapCache) Close() {}
