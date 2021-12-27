# LazyLRU: An in-memory cache with limited locking

Build status: [![Build status](https://badge.buildkite.com/ad7c5afa9718790714c46a0dbf44ff8cb72ebdb7dcc5e84fb7.svg?branch=master)](https://buildkite.com/bluecore-inc/lazylru) [![Coverage Status](https://coveralls.io/repos/github/TriggerMail/lazylru/badge.svg?branch=master)](https://coveralls.io/github/TriggerMail/lazylru?branch=master)

This is a cache implementation that uses a hash table for lookups and a priority queue to approximate LRU. Approximate because the usage is not updated on every get. Rather, items close to the head of the queue, those most likely to be read again and least likely to age out, are not updated. This assumption does not hold under every condition -- if the cache is undersized and churning a lot, this implementation will perform worse than an LRU that updates on every read.

Read about it on [Medium](https://medium.com/bluecore-engineering/lazylru-laughing-all-the-way-to-production-19d2a053c3cb) or hear about it on the [Alexa's Input podcast](https://anchor.fm/alexagriffith/episodes/Cache-Only-with-Mike-Hurwitz-e146ob2).

## What makes this one different

The reason for the occasional updates is not because the updates themselves are expensive -- the [heap](https://golang.org/pkg/container/heap/)-based implementation keeps it quite cheap. Rather, it is the exclusive locking required during that move that drove this design. The lock is more expensive than the move by a wide margin, but is required in concurrent access scenarios. This implementation uses an [RWMutex](https://golang.org/pkg/sync/#RWMutex), but only takes out the exclusive write lock when moving or inserting elements.

Several other LRU caches found on the internet use a doubly-linked list for the recency list. That has the advantage of requiring only four pointer rewrites. By contrast, the heap implementation may require as many as log(n) pointer rewrites. However, the linked-list approach cannot keep track of the position of each node in the list, so the optimization attempted here (ignoring the first quarter of the list) is not possible. Theoretically, the array-based heap implementation also provides better locality, but I don't think I can definitively prove that one way or the other.

There are deletes on read, as well as asynchronous deletes on a timer. When the ticker fires, it will pick a random spot in the heap and check the next 100 items for expiry. Once those 100 items are checked, if any should be deleted, an exlcusive lock will be taken and all items deleted in that single lock cycle, hopefully reducing the cost per deleted item.

All LRU implementations need to update records based on their insert or last access. This is usually handled using time. Time, however, can lead to collisions and undefined behavior. For simplicity, Lazy LRU uses an atomic counter, which means that every insert/update has a unique, integral value.

## Features

* Thread-safe
* Tunable cache size
* Constant-time reads without an exclusive lock
* O(log n) inserts
* Built-in expiry, including purging expired items in the background
* Nearly 100% test coverage

## Non-features

### Deterministic reaping

An optimization that was considered was to keep the inserted values in a second queue (linked list or ring buffer), based on the time of their expiry. While this has the advantage of making the search for expired items extremely cheap, the complexity of maintaining a third reference to each data item seemed like more trouble than it was worth. I may change my mind later.

### Sharding

This is a big one. Lots of cache implemetations get around the lock contention issues by sharding the key space. LazyLRU does not _prevent_ that, but it doesn't do it either. The lack of exclusive locks under the most common reading circumstances should reduce the need to shard, though that really depends on your use cases.

If sharding makes sense for you, it should be pretty easy to make a list of LazyLRU instances, hash your keys before reading or writing to select your cache instance, and go from there. As many LazyLRU instances as you want can coexist a single process. If sharded caches are the path to living as your most authentic self, LazyLRU won't keep your peacock caged.

## Usage

### Go &lt;= 1.17

Like Go's [`heap`](https://golang.org/pkg/container/heap/) itself, Lazy LRU uses the `interface{}` type for its values. That means that casting is required on the way out. I promised that as soon as Go had [generics](https://go.googlesource.com/proposal/+/master/design/go2draft-contracts.md), I'd get right on it. See below!

```go
// import "github.com/TriggerMail/lazylru"

lru := lazylru.New(10, 5 * time.minute)
defer lru.Close()

lru.Set("abloy", "medeco")

v, ok := lru.Get("abloy")
vstr, vok := v.(string)
```

### Go 1.18 (beta)

The Go [`heap`](https://golang.org/pkg/container/heap/) has been copied and made to support generics. That allows the LRU to also support generics. To access that feature, import the `lazylru/generic` module. To maintain compatibility, the `New` factory method still uses `string` keys and `interface{}` values. However, this is just a wrapper over the `NewT[K,V]` factory method.

Once Go 1.18 is released, baked in, commonly used, etc, the `lazylru/generic` module will probably be retired and only the `lazylru` module will remain. Because the `New` factory method is the same, the changes here are purely additive and are in the spirit of the [compatibility guarantee](https://go.dev/doc/go1compat).

```go
// import "github.com/TriggerMail/lazylru/generic"

lru := lazylru.NewT[string, string](10, 5 * time.minute)
defer lru.Close()

lru.Set("abloy", "medeco")

vstr, ok := lru.Get("abloy")
```

It is important to note that `LazyLRU` should be closed if the TTL is non-zero. Otherwise, the background reaper thread will be left running. To be fair, under most circumstances I can imagine, the cache lives as long as the host process. So do what you like.
