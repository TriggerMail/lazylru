package lazylru

import (
	"time"

	heap "github.com/TriggerMail/lazylru/containers/heap"
)

// An item is something we manage in a insertNumber queue.
// The index is needed by update and is maintained by the heap.Interface methods.
type item[K any, V any] struct {
	expiration   time.Time
	value        V
	key          K
	insertNumber uint64
	index        int
}

// itemPQ isn't thread safe, so it is the responsibility of the containing
// LazyLRU to be safe in the face of concurrent access
type itemPQ[K any, V any] []*item[K, V]

func (pq itemPQ[K, V]) Len() int { return len(pq) }

func (pq itemPQ[K, V]) Less(i, j int) bool {
	// We want Pop to give us the lowest, not highest insertNumber so we use less than here.
	return pq[i].insertNumber < pq[j].insertNumber
}

func (pq itemPQ[K, V]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *itemPQ[K, V]) Push(pqi *item[K, V]) {
	n := len(*pq)
	pqi.index = n
	*pq = append(*pq, pqi)
}

func (pq *itemPQ[K, V]) Pop() *item[K, V] {
	if len(*pq) == 0 {
		return nil
	}
	old := *pq
	n := len(old)
	pqi := old[n-1]
	old[n-1] = nil // avoid memory leak
	pqi.index = -1 // for safety
	*pq = old[0 : n-1]
	return pqi
}

// update modifies the insertNumber and value of an item in the queue.
func (pq *itemPQ[K, V]) update(pqi *item[K, V], insertNumber uint64) {
	pqi.insertNumber = insertNumber
	heap.Fix[*item[K, V]](pq, pqi.index)
}
