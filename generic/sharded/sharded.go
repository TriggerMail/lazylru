package sharded

import (
	"errors"
	"time"

	lazylru "github.com/TriggerMail/lazylru/generic"
)

// LazyLRU is a sharded version of the lazylru.LazyLRU cache. The goal of this
// cache is to reduce lock contention via sharding. This may have a negative
// impact on memory locality, so your mileage may vary.
//
// LazyLRU is an LRU cache that only reshuffles values if it is somewhat full.
// This is a cache implementation that uses a hash table for lookups and a
// priority queue to approximate LRU. Approximate because the usage is not
// updated on every get. Rather, items close to the head of the queue, those
// most likely to be read again and least likely to age out, are not updated.
// This assumption does not hold under every condition -- if the cache is
// undersized and churning a lot, this implementation will perform worse than an
// LRU that updates on every read.
type LazyLRU[K comparable, V any] struct {
	sharder func(K) uint64
	shards  []*lazylru.LazyLRU[K, V]
	ttl     time.Duration
}

// New creates a new sharded cache with strings for keys and any (interface{})
// for values
//
// Deprecated: To avoid the casting, use the generic NewT interface instead
func New(maxItemsPerShard int, ttl time.Duration, numShards int) *LazyLRU[string, any] {
	return NewT[string, any](maxItemsPerShard, ttl, numShards, StringSharder)
}

// NewT creates a new sharded cache. The sharder function must be consistent
// and should be as uniformly-distributed over the expected source keys as
// possible. For string and []byte keys, the pre-canned StringSharder and
// BytesSharder are appropriate. These are both based on the HashingSharder,
// which callers can use to create sharder functions for custom types.
func NewT[K comparable, V any](maxItemsPerShard int, ttl time.Duration, numShards int, sharder func(K) uint64) *LazyLRU[K, V] {
	shards := make([]*lazylru.LazyLRU[K, V], numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = lazylru.NewT[K, V](maxItemsPerShard, ttl)
	}

	return &LazyLRU[K, V]{sharder, shards, ttl}
}

// ShardIx determines the target shard for the provided key
func (slru *LazyLRU[K, V]) ShardIx(key K) int {
	return int(slru.sharder(key)&0x7FFFFFFFFFFFFFFF) % len(slru.shards)
}

// IsRunning indicates whether the background reaper is active on at least one
// of the shards
func (slru *LazyLRU[K, V]) IsRunning() bool {
	for _, s := range slru.shards {
		if s.IsRunning() {
			return true
		}
	}
	return false
}

// Reap removes all expired items from the cache
func (slru *LazyLRU[K, V]) Reap() {
	for _, s := range slru.shards {
		s.Reap()
	}
}

// Get retrieves a value from the cache. The returned bool indicates whether the
// key was found in the cache.
func (slru *LazyLRU[K, V]) Get(key K) (V, bool) {
	return slru.shards[slru.ShardIx(key)].Get(key)
}

// MGet retrieves values from the cache. Missing values will not be returned.
func (slru *LazyLRU[K, V]) MGet(keys ...K) map[K]V {
	retval := map[K]V{}
	if len(keys) == 0 {
		return retval
	}
	if len(keys) == 1 {
		v, ok := slru.Get(keys[0])
		if ok {
			retval[keys[0]] = v
		}
		return retval
	}
	shardMapper := newKeyShardHelper(keys, slru.ShardIx)
	for {
		shardIx, skeys := shardMapper.TakeGroup()
		if shardIx < 0 {
			break
		}
		for k, v := range slru.shards[shardIx].MGet(skeys...) {
			retval[k] = v
		}
	}
	return retval
}

// Set writes to the cache
func (slru *LazyLRU[K, V]) Set(key K, value V) {
	slru.shards[slru.ShardIx(key)].Set(key, value)
}

// SetTTL writes to the cache, expiring with the given time-to-live value
func (slru *LazyLRU[K, V]) SetTTL(key K, value V, ttl time.Duration) {
	slru.shards[slru.ShardIx(key)].SetTTL(key, value, ttl)
}

// MSet writes multiple keys and values to the cache. If the "key" and "value"
// parameters are of different lengths, this method will return an error.
func (slru *LazyLRU[K, V]) MSet(keys []K, values []V) error {
	return slru.MSetTTL(keys, values, slru.ttl)
}

// MSetTTL writes multiple keys and values to the cache, expiring with the given
// time-to-live value. If the "key" and "value" parameters are of different
// lengths, this method will return an error.
func (slru *LazyLRU[K, V]) MSetTTL(keys []K, values []V, ttl time.Duration) error {
	// we don't need to store stuff that is already expired
	if ttl <= 0 {
		return nil
	}
	if len(keys) != len(values) {
		return errors.New("Mismatch between number of keys and number of values")
	}

	if len(keys) == 0 {
		return nil
	}
	if len(keys) == 1 {
		slru.SetTTL(keys[0], values[0], ttl)
		return nil
	}
	shardMapper := newKVShardHelper(keys, values, slru.ShardIx)
	for {
		shardIx, skeys, svals := shardMapper.TakeGroup()
		if shardIx < 0 {
			break
		}
		if err := slru.shards[shardIx].MSetTTL(skeys, svals, ttl); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the number of items in the cache
func (slru *LazyLRU[K, V]) Len() int {
	retval := 0
	for _, s := range slru.shards {
		retval += s.Len()
	}
	return retval
}

// Close stops the reaper process. This is safe to call multiple times.
func (slru *LazyLRU[K, V]) Close() {
	for _, s := range slru.shards {
		s.Close()
	}
}

// Stats gets a copy of the stats held by the cache. Note that this is a copy,
// so returned objects will not update as the service continues to execute. The
// returned value is a sum of each statistic across all shards.
func (slru *LazyLRU[K, V]) Stats() lazylru.Stats {
	stats := slru.shards[0].Stats()
	for i := 1; i < len(slru.shards); i++ {
		lstats := slru.shards[i].Stats()
		stats.KeysWritten += lstats.KeysWritten
		stats.KeysReadOK += lstats.KeysReadOK
		stats.KeysReadNotFound += lstats.KeysReadNotFound
		stats.KeysReadExpired += lstats.KeysReadExpired
		stats.Shuffles += lstats.Shuffles
		stats.Evictions += lstats.Evictions
		stats.KeysReaped += lstats.KeysReaped
		stats.ReaperCycles += lstats.ReaperCycles
	}
	return stats
}
