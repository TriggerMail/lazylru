package lazylru

import (
	"testing"

	heap "github.com/TriggerMail/lazylru/containers/heap"

	"github.com/stretchr/testify/require"
)

func TestPopEmpty(t *testing.T) {
	pq := itemPQ[string, int]{}
	require.Nil(t, pq.Pop())
}

func TestPushPop(t *testing.T) {
	pq := itemPQ[string, int]{}

	heap.Push[*item[string, int]](&pq, &item[string, int]{
		value:        13,
		insertNumber: 0,
		key:          "schlage",
	})
	pqi := heap.Pop[*item[string, int]](&pq)
	require.Equal(t, "schlage", pqi.key)
	require.Equal(t, 13, pqi.value)
}

func TestPushPopOrdered(t *testing.T) {
	pq := itemPQ[string, int]{}

	heap.Push[*item[string, int]](&pq, &item[string, int]{
		value:        13,
		insertNumber: 0,
		key:          "schlage",
	})
	heap.Push[*item[string, int]](&pq, &item[string, int]{
		value:        13,
		insertNumber: 1,
		key:          "kwikset",
	})
	heap.Push[*item[string, int]](&pq, &item[string, int]{
		value:        13,
		insertNumber: 2,
		key:          "abloy",
	})

	require.Equal(t, "schlage", heap.Pop[*item[string, int]](&pq).key)
	require.Equal(t, "kwikset", heap.Pop[*item[string, int]](&pq).key)
	require.Equal(t, "abloy", heap.Pop[*item[string, int]](&pq).key)
}

func TestPushPopUpdate(t *testing.T) {
	pq := itemPQ[string, int]{}

	heap.Push[*item[string, int]](&pq, &item[string, int]{
		value:        13,
		insertNumber: 0,
		key:          "schlage",
	})
	heap.Push[*item[string, int]](&pq, &item[string, int]{
		value:        13,
		insertNumber: 2,
		key:          "abloy",
	})
	kwi := &item[string, int]{
		value:        13,
		insertNumber: 1,
		key:          "kwikset",
	}
	heap.Push[*item[string, int]](&pq, kwi)
	pq.update(kwi, 3)

	require.Equal(t, "schlage", heap.Pop[*item[string, int]](&pq).key)
	require.Equal(t, "abloy", heap.Pop[*item[string, int]](&pq).key)
	require.Equal(t, "kwikset", heap.Pop[*item[string, int]](&pq).key)
}
