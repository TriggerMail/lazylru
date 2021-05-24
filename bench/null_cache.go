package main

type nullCache struct{}

var NullCache = nullCache{}

func (nc nullCache) Get(key string) (interface{}, bool) {
	return nil, false
}

func (nc nullCache) Set(key string, value interface{}) {}

func (nc nullCache) Close() {}
