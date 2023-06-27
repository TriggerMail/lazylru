package lazylru_test

import (
	"testing"
	"time"

	lazylru "github.com/TriggerMail/lazylru"
	"github.com/stretchr/testify/require"
)

func FuzzIntInt(f *testing.F) {
	lru := lazylru.NewT[int, int](1000, 10*time.Second)
	f.Cleanup(lru.Close)
	f.Add(0, 0)
	f.Fuzz(func(t *testing.T, k int, v int) {
		lru.Set(k, v)

		a, ok := lru.Get(k)
		require.True(t, ok, "failed to find %d", k)
		require.Equal(t, v, a, "unexpected value for prev %d", k)
		require.GreaterOrEqual(t, 1000, lru.Len())
	})
}

func FuzzStringStruct(f *testing.F) {
	type data struct {
		val int
	}

	lru := lazylru.NewT[string, *data](1000, 10*time.Second)
	f.Cleanup(lru.Close)
	f.Add("foo", 0)
	f.Fuzz(func(t *testing.T, k string, v int) {
		lru.Set(k, &data{v})
		a, ok := lru.Get(k)
		require.True(t, ok)
		require.Equal(t, v, a.val)
		require.GreaterOrEqual(t, 1000, lru.Len())
	})
}
