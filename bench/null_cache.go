package main

type nullCache struct{}

// NullCache is a cache that holds nothing and returns nothing
var NullCache = nullCache{}

// Get always returns `nil, false“
func (nc nullCache) Get(key string) (string, bool) {
	return "", false
}

// Set does nothing
func (nc nullCache) Set(key string, value string) {}

// Close does nothing
func (nc nullCache) Close() {}
