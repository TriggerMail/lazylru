package lazylru

import (
	"container/heap"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Stats struct {
	KeysWritten      uint32
	KeysReadOK       uint32
	KeysReadNotFound uint32
	KeysReadExpired  uint32
	Shuffles         uint32
	Evictions        uint32
	KeysReaped       uint32
	ReaperCycles     uint32
}

type LazyLRU struct {
	items     itemPQ
	index     map[string]*item
	maxItems  int
	itemIx    uint64
	lock      sync.RWMutex
	ttl       time.Duration
	doneCh    chan int
	isRunning bool
	isClosing bool
	stats     Stats
}

// New creates a LazyLRU with the given capacity and default expiration. If
// maxItems is zero or fewer, the cache will not hold anything, but does still
// incur some runtime penalties. If ttl is greater than zero, a background
// ticker will be engaged to proactively remove expired items.
func New(maxItems int, ttl time.Duration) *LazyLRU {
	if maxItems < 0 {
		maxItems = 0
	}

	doneCh := make(chan int)
	lru := &LazyLRU{
		items:     itemPQ{},
		index:     map[string]*item{},
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

// IsRunning indicates whether the background reaper is active
func (lru *LazyLRU) IsRunning() bool {
	lru.lock.RLock()
	defer lru.lock.RUnlock()
	return lru.isRunning
}

// reaper engages a background goroutine to randomly select items from the list
// on a regular basis and check them for expiry. This does not check the whole
// list, but starts at a random point, looking for expired items.
func (lru *LazyLRU) reaper() {
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
			deathList := make([]*item, 0, 100)
			keepGoing := true
			for keepGoing {
				select {
				case <-lru.doneCh:
					lru.lock.Lock()
					lru.items = nil
					lru.index = nil
					lru.maxItems = 0
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
func (lru *LazyLRU) Reap() {
	lru.reap(0, make([]*item, 0, 100))
}

func (lru *LazyLRU) reap(start int, deathList []*item) {
	timestamp := time.Now()
	if lru.Len() == 0 {
		return
	}

	cycles := uint32(0)
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
			start = rand.Intn(len(lru.items))
		}
		end := start + 100 // why 100? no idea
		if end > len(lru.items) {
			end = len(lru.items)
		}
		for i := start; i < end; i++ {
			if lru.items[i].expiration.Before(timestamp) {
				deathList = append(deathList, lru.items[i])
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
			_ = heap.Pop(&lru.items)
		}
		lru.lock.Unlock()
	}
	atomic.AddUint32(&lru.stats.ReaperCycles, cycles)
}

// shouldBubble determines if a particualr item should be updated on read and
// moved to the end of the queue. This is NOT thread safe and should only be
// called with a lock in place.
func (lru *LazyLRU) shouldBubble(index int) bool {
	return (index + (lru.maxItems - lru.items.Len())) < (lru.maxItems >> 2)
}

// Get retrieves a value from the cache. The returned bool indicates whether the
// key was found in the cache.
func (lru *LazyLRU) Get(key string) (interface{}, bool) {
	lru.lock.RLock()
	pqi, ok := lru.index[key]
	lru.lock.RUnlock()
	if !ok {
		atomic.AddUint32(&lru.stats.KeysReadNotFound, 1)
		return nil, false
	}

	// there is a dangerous case if the read/lock/read pattern returns an
	// unexpired key on the second read -- if we are not careful, we may end up
	// trying to take the lock twice. Because "defer" can't help us here, I'm
	// being really explicit about whether or not we have the lock already.
	locked := false
	// if the item is expired, remove it
	if pqi.expiration.Before(time.Now()) && pqi.index >= 0 {
		lru.lock.Lock()
		locked = true

		// double check in case this has already been removed
		if pqi.expiration.Before(time.Now()) && pqi.index >= 0 {
			// this will push the item to the end
			lru.items.update(pqi, 0)
			delete(lru.index, pqi.key)
			// cut off all the expired items. should only be one
			for lru.items.Len() > 0 && lru.items[0].insertNumber == 0 {
				_ = heap.Pop(&lru.items)
			}
			lru.stats.KeysReadExpired++
			lru.lock.Unlock()
			return nil, false
		}
	}

	// we only want to shuffle this item if it is far
	// enough from the front that it is at risk of being
	// evicted. This will save us from locking 75% of
	// the time
	if lru.shouldBubble(pqi.index) {
		if !locked {
			lru.lock.Lock()
			locked = true
		}
		// double check because someone else may have shuffled
		if lru.shouldBubble(pqi.index) {
			lru.items.update(pqi, atomic.AddUint64(&(lru.itemIx), 1))
			lru.stats.Shuffles++
		}
	}
	if locked {
		lru.lock.Unlock()
	}
	atomic.AddUint32(&lru.stats.KeysReadOK, 1)
	return pqi.value, ok
}

// MGet retrieves values from the cache. Missing values will not be returned.
func (lru *LazyLRU) MGet(keys ...string) map[string]interface{} {
	retval := make(map[string]interface{}, len(keys))
	maybeExpired := make([]string, 0, len(keys))
	needsShuffle := make([]string, 0, len(keys))

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
		_ = heap.Pop(&lru.items)
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
func (lru *LazyLRU) Set(key string, value interface{}) {
	lru.SetTTL(key, value, lru.ttl)
}

// SetTTL writes to the cache, expiring with the given time-to-live value
func (lru *LazyLRU) SetTTL(key string, value interface{}, ttl time.Duration) {
	lru.lock.Lock()
	lru.setInternal(key, value, time.Now().Add(ttl))
	lru.lock.Unlock()
}

// setInternal writes elements. This is NOT thread safe and should always be
// called with a write lock
func (lru *LazyLRU) setInternal(key string, value interface{}, expiration time.Time) {
	lru.stats.KeysWritten++
	if pqi, ok := lru.index[key]; ok {
		pqi.expiration = expiration
		pqi.value = value
		lru.items.update(pqi, atomic.AddUint64(&(lru.itemIx), 1))
	} else {
		pqi := &item{
			value:        value,
			insertNumber: atomic.AddUint64(&(lru.itemIx), 1),
			key:          key,
			expiration:   expiration,
		}

		// remove excess
		for lru.items.Len() >= lru.maxItems {
			deadGuy := heap.Pop(&lru.items).(*item)
			delete(lru.index, deadGuy.key)
			lru.stats.Evictions++
		}
		heap.Push(&lru.items, pqi)
		lru.index[key] = pqi
	}
}

// MSet writes multiple keys and values to the cache. If the "key" and "value"
// parameters are of different lengths, this method will return an error.
func (lru *LazyLRU) MSet(keys []string, values []interface{}) error {
	return lru.MSetTTL(keys, values, lru.ttl)
}

// MSetTTL writes multiple keys and values to the cache, expiring with the given
// time-to-live value. If the "key" and "value" parameters are of different
// lengths, this method will return an error.
func (lru *LazyLRU) MSetTTL(keys []string, values []interface{}, ttl time.Duration) error {
	// we don't need to store stuff that is already expired
	if ttl < 0 {
		return nil
	}
	if len(keys) != len(values) {
		return errors.New("Mismatch between number of keys and number of values")
	}

	lru.lock.Lock()
	expiration := time.Now().Add(ttl)
	for i := 0; i < len(keys); i++ {
		lru.setInternal(keys[i], values[i], expiration)
	}
	lru.lock.Unlock()
	return nil
}

// Len returns the number of items in the cache
func (lru *LazyLRU) Len() int {
	lru.lock.RLock()
	defer lru.lock.RUnlock()
	return len(lru.items)
}

// Close stops the reaper process. This is safe to call multiple times.
func (lru *LazyLRU) Close() {
	lru.lock.Lock()
	if !lru.isClosing {
		close(lru.doneCh)
		lru.isClosing = true
	}
	lru.lock.Unlock()
}

// Stats gets a copy of the stats held by the cache. Note that this is a copy,
// so returned objects will not update as the service continues to execute.
func (lru *LazyLRU) Stats() Stats {
	// note that this returns a copy of stats, not a reference
	return lru.stats
}
