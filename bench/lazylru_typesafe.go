package main

import "github.com/TriggerMail/lazylru"

// LazyLRUTypesafe is a typesafe wraper for the interface-based LazyLRU
type LazyLRUTypesafe[V any] lazylru.LazyLRU

// Get retrieves an item from the cache
func (c *LazyLRUTypesafe[V]) Get(key string) (V, bool) {
	var retval V
	v, ok := (*lazylru.LazyLRU)(c).Get(key)
	if ok {
		retval, ok = v.(V)
	}
	return retval, ok
}

// Set writes an item to the cache
func (c *LazyLRUTypesafe[V]) Set(key string, value V) {
	(*lazylru.LazyLRU)(c).Set(key, value)
}

// Close stops the reaper process. This is safe to call multiple times.
func (c *LazyLRUTypesafe[V]) Close() {
	(*lazylru.LazyLRU)(c).Close()
}
