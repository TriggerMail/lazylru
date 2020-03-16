package lazylru

import (
	"container/heap"
	"time"
)

// An item is something we manage in a insertNumber queue.
type item struct {
	value        interface{} // The value of the item; arbitrary.
	insertNumber uint64      // The insertNumber is used for priority (age)
	// The index is needed by update and is maintained by the heap.Interface methods.
	index      int // The index of the item in the heap.
	key        string
	expiration time.Time
}

// itemPQ isn't thread safe, so it is the responsibility of the containing
// LazyLRU to be safe in the face of concurrent access
type itemPQ []*item

func (pq itemPQ) Len() int { return len(pq) }

func (pq itemPQ) Less(i, j int) bool {
	// We want Pop to give us the lowest, not highest insertNumber so we use less than here.
	return pq[i].insertNumber < pq[j].insertNumber
}

func (pq itemPQ) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *itemPQ) Push(x interface{}) {
	n := len(*pq)
	pqi := x.(*item)
	pqi.index = n
	*pq = append(*pq, pqi)
}

func (pq *itemPQ) Pop() interface{} {
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
func (pq *itemPQ) update(pqi *item, insertNumber uint64) {
	pqi.insertNumber = insertNumber
	heap.Fix(pq, pqi.index)
}
