package lazylru

import (
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	heap "github.com/TriggerMail/lazylru/containers/heap"
)

// EvictCB is a callback function that will be executed when items are removed
// from the cache via eviction due to max size or because the TTL has been
// exceeded. These functions will not be called with a lock and will not block
// future reaping. Be sure any callback registered can complete the number of
// expected calls (based on your expire/eviction rates) or you may create a
// backlog of goroutines.
type EvictCB[K comparable, V any] func(K, V)

// LazyLRU is an LRU cache that only reshuffles values if it is somewhat full.
// This is a cache implementation that uses a hash table for lookups and a
// priority queue to approximate LRU. Approximate because the usage is not
// updated on every get. Rather, items close to the head of the queue, those
// most likely to be read again and least likely to age out, are not updated.
// This assumption does not hold under every condition -- if the cache is
// undersized and churning a lot, this implementation will perform worse than an
// LRU that updates on every read.
type LazyLRU[K comparable, V any] struct {
	onEvict    []EvictCB[K, V]
	doneCh     chan int
	index      map[K]*item[K, V]
	items      itemPQ[K, V]
	maxItems   int
	itemIx     uint64
	ttl        time.Duration
	stats      Stats
	lock       sync.RWMutex
	isRunning  bool
	isClosing  bool
	numEvictCB atomic.Int32 // faster to check than locking and checking the length of onEvict
}

// New creates a LazyLRU[string, interface{} with the given capacity and default
// expiration. This is compatible with the pre-generic interface. The generic
// version is available as `NewT`. If maxItems is zero or fewer, the cache will
// not hold anything, but does still incur some runtime penalties. If ttl is
// greater than zero, a background ticker will be engaged to proactively remove
// expired items.
//
// Deprecated: To avoid the casting, use the generic NewT interface instead
func New(maxItems int, ttl time.Duration) *LazyLRU[string, interface{}] {
	return NewT[string, interface{}](maxItems, ttl)
}

// NewT creates a LazyLRU with the given capacity and default expiration. If
// maxItems is zero or fewer, the cache will not hold anything, but does still
// incur some runtime penalties. If ttl is greater than zero, a background
// ticker will be engaged to proactively remove expired items.
func NewT[K comparable, V any](maxItems int, ttl time.Duration) *LazyLRU[K, V] {
	if maxItems < 0 {
		maxItems = 0
	}

	doneCh := make(chan int)
	lru := &LazyLRU[K, V]{
		items:     itemPQ[K, V]{},
		index:     map[K]*item[K, V]{},
		maxItems:  maxItems,
		itemIx:    1, // starting at 1 means that 0 can always be popped
		ttl:       ttl,
		doneCh:    doneCh,
		isRunning: false,
		stats:     Stats{},
	}

	if ttl > 0 {
		lru.reaper()
	} else {
		lru.isClosing = true
		close(doneCh)
	}

	return lru
}

// OnEvict registers a callback that will be executed when items are removed
// from the cache via eviction due to max size or because the TTL has been
// exceeded. These functions will not be called with a lock and will not block
// future reaping. Be sure any callback registered can complete the number of
// expected calls (based on your expire/eviction rates) or you may create a
// backlog of goroutines.
//
// If a Set or MSet operation causes an eviction, this function will be called
// synchronously to that Set or MSet call.
func (lru *LazyLRU[K, V]) OnEvict(cb EvictCB[K, V]) {
	lru.lock.Lock()
	lru.onEvict = append(lru.onEvict, cb)
	lru.numEvictCB.Add(1)
	lru.lock.Unlock()
}

func (lru *LazyLRU[K, V]) execOnEvict(deathList []*item[K, V]) {
	if len(deathList) == 0 {
		return
	}
	if lru.numEvictCB.Load() == 0 {
		return
	}

	var callbacks []EvictCB[K, V]
	lru.lock.RLock()
	callbacks = lru.onEvict
	lru.lock.RUnlock()
	if len(callbacks) == 0 {
		return // this should never happen
	}

	for _, item := range deathList {
		for _, cb := range callbacks {
			cb(item.key, item.value)
		}
	}
}

// IsRunning indicates whether the background reaper is active
func (lru *LazyLRU[K, V]) IsRunning() bool {
	lru.lock.RLock()
	defer lru.lock.RUnlock()
	return lru.isRunning
}

// reaper engages a background goroutine to randomly select items from the list
// on a regular basis and check them for expiry. This does not check the whole
// list, but starts at a random point, looking for expired items.
func (lru *LazyLRU[K, V]) reaper() {
	if lru.ttl > 0 {
		watchTime := lru.ttl / 10
		if watchTime < time.Millisecond {
			watchTime = time.Millisecond
		}
		if watchTime > time.Second {
			watchTime = time.Second
		}
		ticker := time.NewTicker(watchTime)
		lru.lock.Lock()
		lru.isRunning = true
		lru.lock.Unlock()
		go func() {
			deathList := make([]*item[K, V], 0, 100)
			keepGoing := true
			for keepGoing {
				select {
				case <-lru.doneCh:
					lru.lock.Lock()
					// These triggered a race with the shouldBubble method. It
					// shouldn't really matter, but there isn't much reason to
					// worry about these things when the whole thing is going
					// away. Putting a read lock around that first shouldBubble
					// call had an 8.5% penalty on the read path, so leaving the
					// data behind seemed like the better choice.
					// Interestingly, the non-generic version of this code did
					// not trigger the race condition.
					// lru.items = nil
					// lru.index = nil
					// lru.maxItems = 0
					lru.isRunning = false
					lru.lock.Unlock()
					keepGoing = false
					break
				case <-ticker.C:
					lru.reap(-1, deathList)
				}
			}
			ticker.Stop()
		}()
	}
}

// Reap removes all expired items from the cache
func (lru *LazyLRU[K, V]) Reap() {
	lru.reap(0, make([]*item[K, V], 0, 100))
}

func (lru *LazyLRU[K, V]) reap(start int, deathList []*item[K, V]) {
	timestamp := time.Now()
	if lru.Len() == 0 {
		return
	}

	cycles := uint32(0)
	var aggDeathList []*item[K, V]
	for {
		cycles++
		// grab a read lock while we are looking for items to kill
		lru.lock.RLock()

		// make sure there is nothing left from the last cycle
		deathList = deathList[:0]
		if (!lru.isRunning) || len(lru.items) == 0 {
			lru.lock.RUnlock()
			break
		}
		if start < 0 {
			start = rand.IntN(len(lru.items)) //nolint:gosec
		}
		end := start + 100 // why 100? no idea
		if end > len(lru.items) {
			end = len(lru.items)
		}
		for i := start; i < end; i++ {
			if lru.items[i].expiration.Before(timestamp) {
				deathList = append(deathList, lru.items[i])
				aggDeathList = append(aggDeathList, lru.items[i])
			}
		}
		lru.lock.RUnlock()

		// if there are no candidates to kill, we're done
		// break is safe here because we are between locks
		if len(deathList) == 0 {
			break
		}

		lru.lock.Lock()
		// mark the expired candidates as dead, remove from index
		for ix, pqi := range deathList {
			// it may have been touched between the locks
			if pqi.insertNumber > 0 && pqi.expiration.Before(timestamp) {
				lru.items.update(pqi, 0)
				delete(lru.index, pqi.key)
				deathList[ix] = nil
				lru.stats.KeysReaped++
			}
		}
		// cut off all the expired items
		for 0 < lru.items.Len() && lru.items[0].insertNumber == 0 {
			_ = heap.Pop[*item[K, V]](&lru.items)
		}
		lru.lock.Unlock()
	}
	atomic.AddUint32(&lru.stats.ReaperCycles, cycles)
	if len(aggDeathList) > 0 && lru.numEvictCB.Load() > 0 {
		lru.execOnEvict(aggDeathList)
	}
}

// shouldBubble determines if a particular item should be updated on read and
// moved to the end of the queue. This is NOT thread safe and should only be
// called with a lock in place.
func (lru *LazyLRU[K, V]) shouldBubble(index int) bool {
	return (index + (lru.maxItems - lru.items.Len())) < (lru.maxItems >> 2)
}

// Get retrieves a value from the cache. The returned bool indicates whether the
// key was found in the cache.
func (lru *LazyLRU[K, V]) Get(key K) (V, bool) {
	lru.lock.RLock()
	// pqi may be touched between when we release this lock and the writer lock
	// below, so we need to store the value we read in the stack before checking
	// the expiration and such. It won't hurt anything because we will take a
	// write lock and check pqi again, but it's the right thing to do and makes
	// the race detector happy.
	pqi, ok := lru.index[key]
	if !ok {
		lru.lock.RUnlock()
		atomic.AddUint32(&lru.stats.KeysReadNotFound, 1)
		var zero V
		return zero, false
	}
	qi := *pqi
	lru.lock.RUnlock()

	// there is a dangerous case if the read/lock/read pattern returns an
	// unexpired key on the second read -- if we are not careful, we may end up
	// trying to take the lock twice. Because "defer" can't help us here, I'm
	// being really explicit about whether or not we have the lock already.
	var locked bool
	// if the item is expired, remove it
	if qi.expiration.Before(time.Now()) && qi.index >= 0 {
		lru.lock.Lock()
		locked = true

		// double check in case this has already been removed
		if pqi.expiration.Before(time.Now()) && pqi.index >= 0 {
			// this will push the item to the end
			lru.items.update(pqi, 0)
			delete(lru.index, pqi.key)
			// cut off all the expired items. should only be one
			for lru.items.Len() > 0 && lru.items[0].insertNumber == 0 {
				_ = heap.Pop[*item[K, V]](&lru.items)
			}
			lru.stats.KeysReadExpired++
			lru.lock.Unlock()
			var zero V
			return zero, false
		}
	}

	// We only want to shuffle this item if it is far enough from the front that
	// it is at risk of being evicted. This will save us from exclusive locking
	// 75% of the time.
	if !locked {
		lru.lock.RLock()
		maybeShould := lru.shouldBubble(pqi.index)
		lru.lock.RUnlock()
		if !maybeShould {
			atomic.AddUint32(&lru.stats.KeysReadOK, 1)
			return qi.value, ok
		}
	}

	if !locked {
		lru.lock.Lock()
		// locked = true  // ineffectual
	}
	// double check because someone else may have shuffled
	if lru.shouldBubble(pqi.index) {
		lru.items.update(pqi, atomic.AddUint64(&(lru.itemIx), 1))
		lru.stats.Shuffles++
	}

	lru.lock.Unlock() // we will definitely be locked if we got here

	atomic.AddUint32(&lru.stats.KeysReadOK, 1)
	return qi.value, ok
}

// MGet retrieves values from the cache. Missing values will not be returned.
func (lru *LazyLRU[K, V]) MGet(keys ...K) map[K]V {
	retval := make(map[K]V, len(keys))
	maybeExpired := make([]K, 0, len(keys))
	needsShuffle := make([]K, 0, len(keys))

	lru.lock.RLock()
	notfound := uint32(0)
	for _, key := range keys {
		if pqi, found := lru.index[key]; found {
			retval[key] = pqi.value
			if pqi.expiration.Before(time.Now()) && pqi.index >= 0 {
				maybeExpired = append(maybeExpired, key)
			} else if lru.shouldBubble(pqi.index) {
				needsShuffle = append(needsShuffle, key)
			}
		} else {
			notfound++
		}
	}
	lru.lock.RUnlock()
	if notfound > 0 {
		atomic.AddUint32(&lru.stats.KeysReadNotFound, notfound)
	}

	// if we are done, let's be done
	if len(retval) == 0 || (len(maybeExpired) == 0 && len(needsShuffle) == 0) {
		atomic.AddUint32(&lru.stats.KeysReadOK, uint32(len(retval)))
		return retval
	}

	// we're going to have to change _something_
	lru.lock.Lock()
	defer lru.lock.Unlock()
	for _, key := range maybeExpired {
		pqi, ok := lru.index[key]
		if !ok {
			continue
		}
		// if the item is expired, remove it
		if pqi.expiration.Before(time.Now()) && pqi.index >= 0 {
			// this will push the item to the end
			lru.items.update(pqi, 0)
			delete(lru.index, key)
			delete(retval, key)
			lru.stats.KeysReadExpired++
		}
	}

	// cut off all the expired items
	for lru.items.Len() > 0 && lru.items[0].insertNumber == 0 {
		_ = heap.Pop[*item[K, V]](&lru.items)
	}

	for _, key := range needsShuffle {
		// we only want to shuffle this item if it is far
		// enough from the front that it is at risk of being
		// evicted. This will save us from locking 75% of
		// the time
		pqi, ok := lru.index[key]
		if ok && lru.shouldBubble(pqi.index) {
			lru.stats.Shuffles++
			// double check because someone else may have shuffled
			lru.items.update(pqi, atomic.AddUint64(&(lru.itemIx), 1))
		}
	}

	atomic.AddUint32(&lru.stats.KeysReadOK, uint32(len(retval)))
	return retval
}

// Set writes to the cache
func (lru *LazyLRU[K, V]) Set(key K, value V) {
	lru.SetTTL(key, value, lru.ttl)
}

// SetTTL writes to the cache, expiring with the given time-to-live value
func (lru *LazyLRU[K, V]) SetTTL(key K, value V, ttl time.Duration) {
	lru.lock.Lock()
	deathList := lru.setInternal(key, value, time.Now().Add(ttl))
	lru.lock.Unlock()
	if len(deathList) > 0 && lru.numEvictCB.Load() > 0 {
		lru.execOnEvict(deathList)
	}
}

// setInternal writes elements. This is NOT thread safe and should always be
// called with a write lock
func (lru *LazyLRU[K, V]) setInternal(key K, value V, expiration time.Time) []*item[K, V] {
	if lru.maxItems <= 0 {
		return nil
	}
	var deathList []*item[K, V]
	lru.stats.KeysWritten++
	if pqi, ok := lru.index[key]; ok {
		pqi.expiration = expiration
		pqi.value = value
		lru.items.update(pqi, atomic.AddUint64(&(lru.itemIx), 1))
	} else {
		pqi := &item[K, V]{
			value:        value,
			insertNumber: atomic.AddUint64(&(lru.itemIx), 1),
			key:          key,
			expiration:   expiration,
		}

		// remove excess
		for lru.items.Len() >= lru.maxItems {
			deadGuy := heap.Pop[*item[K, V]](&lru.items)
			delete(lru.index, deadGuy.key)
			deathList = append(deathList, deadGuy)
			lru.stats.Evictions++
		}
		heap.Push[*item[K, V]](&lru.items, pqi)
		lru.index[key] = pqi
	}
	return deathList
}

// MSet writes multiple keys and values to the cache. If the "key" and "value"
// parameters are of different lengths, this method will return an error.
func (lru *LazyLRU[K, V]) MSet(keys []K, values []V) error {
	return lru.MSetTTL(keys, values, lru.ttl)
}

// MSetTTL writes multiple keys and values to the cache, expiring with the given
// time-to-live value. If the "key" and "value" parameters are of different
// lengths, this method will return an error.
func (lru *LazyLRU[K, V]) MSetTTL(keys []K, values []V, ttl time.Duration) error {
	// we don't need to store stuff that is already expired
	if ttl < 0 {
		return nil
	}
	if len(keys) != len(values) {
		return errors.New("Mismatch between number of keys and number of values")
	}

	var deathList []*item[K, V]
	lru.lock.Lock()
	expiration := time.Now().Add(ttl)
	for i := 0; i < len(keys); i++ {
		deathList = append(deathList, lru.setInternal(keys[i], values[i], expiration)...)
	}
	lru.lock.Unlock()
	if len(deathList) > 0 && lru.numEvictCB.Load() > 0 {
		lru.execOnEvict(deathList)
	}
	return nil
}

// Delete elimitates a key from the cache. Removing a key that is not in the index is safe.
func (lru *LazyLRU[K, V]) Delete(key K) {
	// if the key isn't here, don't bother taking the exclusive lock
	lru.lock.RLock()
	_, ok := lru.index[key]
	lru.lock.RUnlock()
	if !ok {
		return
	}
	lru.lock.Lock()
	pqi, ok := lru.index[key]
	if !ok {
		lru.lock.Unlock()
		return
	}
	delete(lru.index, pqi.key)                   // remove from search index
	lru.items.update(pqi, 0)                     // move this item to the top of the heap
	deadguy := heap.Pop[*item[K, V]](&lru.items) // pop item from the top of the heap
	lru.lock.Unlock()
	if lru.numEvictCB.Load() > 0 {
		lru.execOnEvict([]*item[K, V]{deadguy})
	}
}

// Len returns the number of items in the cache
func (lru *LazyLRU[K, V]) Len() int {
	lru.lock.RLock()
	defer lru.lock.RUnlock()
	return len(lru.items)
}

// Close stops the reaper process. This is safe to call multiple times.
func (lru *LazyLRU[K, V]) Close() {
	lru.lock.Lock()
	if !lru.isClosing {
		close(lru.doneCh)
		lru.isClosing = true
	}
	lru.lock.Unlock()
}

// Stats gets a copy of the stats held by the cache. Note that this is a copy,
// so returned objects will not update as the service continues to execute.
func (lru *LazyLRU[K, V]) Stats() Stats {
	// note that this returns a copy of stats, not a reference
	return lru.stats
}
