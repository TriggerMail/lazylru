package lazylru

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPopEmpty(t *testing.T) {
	pq := itemPQ{}
	require.Nil(t, pq.Pop())
}
func TestPushPop(t *testing.T) {
	pq := itemPQ{}

	heap.Push(&pq, &item{
		value:        13,
		insertNumber: 0,
		key:          "schlage",
	})
	pqi := heap.Pop(&pq).(*item)
	require.Equal(t, "schlage", pqi.key)
}

func TestPushPopOrdered(t *testing.T) {
	pq := itemPQ{}

	heap.Push(&pq, &item{
		value:        13,
		insertNumber: 0,
		key:          "schlage",
	})
	heap.Push(&pq, &item{
		value:        13,
		insertNumber: 1,
		key:          "kwikset",
	})
	heap.Push(&pq, &item{
		value:        13,
		insertNumber: 2,
		key:          "abloy",
	})

	require.Equal(t, "schlage", heap.Pop(&pq).(*item).key)
	require.Equal(t, "kwikset", heap.Pop(&pq).(*item).key)
	require.Equal(t, "abloy", heap.Pop(&pq).(*item).key)
}

func TestPushPopUpdate(t *testing.T) {
	pq := itemPQ{}

	heap.Push(&pq, &item{
		value:        13,
		insertNumber: 0,
		key:          "schlage",
	})
	heap.Push(&pq, &item{
		value:        13,
		insertNumber: 2,
		key:          "abloy",
	})
	kwi := &item{
		value:        13,
		insertNumber: 1,
		key:          "kwikset",
	}
	heap.Push(&pq, kwi)
	pq.update(kwi, 3)

	require.Equal(t, "schlage", heap.Pop(&pq).(*item).key)
	require.Equal(t, "abloy", heap.Pop(&pq).(*item).key)
	require.Equal(t, "kwikset", heap.Pop(&pq).(*item).key)
}
