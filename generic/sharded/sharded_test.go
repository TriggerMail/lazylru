package sharded_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/TriggerMail/lazylru/generic/sharded"
	"github.com/stretchr/testify/require"
)

func doShardedTest[K comparable, V any](t *testing.T, maxItems int, ttl time.Duration, numShards int, selector func(K) uint64, test func(t *testing.T, lru *sharded.LazyLRU[K, V]), expected ExpectedStats) {
	lru := sharded.NewT[K, V](maxItems, ttl, numShards, selector)
	test(t, lru)
	lru.Close()
	expected.Test(t, lru.Stats())
}

func TestMakeNew(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, int]) {
		require.NotNil(t, lru)
	},
		ExpectedStats{},
	)
}

func TestGetUnknown(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, int]) {
		v, ok := lru.Get("something new")
		require.Equal(t, 0, v)
		require.False(t, ok)
	},
		ExpectedStats{}.WithKeysReadNotFound(1),
	)
}

func TestGetKnown(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Set("abloy", "medeco")
		v, ok := lru.Get("abloy")
		require.True(t, ok)
		require.Equal(t, "medeco", v)
	},
		ExpectedStats{}.WithKeysWritten(1).WithKeysReadOK(1),
	)
}

func TestMGetUnknown(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		found := lru.MGet("a", "b", "c")
		require.Equal(t, 0, len(found))
	},
		ExpectedStats{}.WithKeysReadNotFound(3),
	)
}

func TestMGetKnown(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet(
			[]string{"abloy", "schlage"},
			[]string{"medeco", "kwikset"},
		)
		require.NoError(t, err)
		found := lru.MGet("abloy", "schlage")
		require.Equal(t, 2, len(found))
		v, ok := found["abloy"]
		require.True(t, ok)
		require.Equal(t, "medeco", v)
	},
		ExpectedStats{}.WithKeysWritten(2).WithKeysReadOK(2),
	)
}

func TestSetNTimes(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		require.Equal(t, 0, lru.Len())
		lru.Set("abloy", "schlage")
		require.Equal(t, 1, lru.Len())
		for i := 0; i < 1000; i++ {
			lru.Set("abloy", "schlage")
		}
		require.Equal(t, 1, lru.Len())
	},
		ExpectedStats{}.WithKeysWritten(1001),
	)
}

func TestMGetOneKnown(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Set("abloy", "medeco")

		found := lru.MGet("abloy")
		require.Equal(t, 1, len(found))

		v, ok := found["abloy"]
		require.True(t, ok)
		require.Equal(t, "medeco", v)
	},
		ExpectedStats{}.WithKeysWritten(1).WithKeysReadOK(1),
	)
}

func TestMSetBad(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet(
			[]string{"abloy"},
			[]string{"medeco", "kwikset"},
		)
		require.Error(t, err)
	},
		ExpectedStats{}.WithKeysWritten(0),
	)
}

func TestMSetTooMany(t *testing.T) {
	doShardedTest(t, 2, time.Hour, 2, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet(
			[]string{"a", "b", "c", "d", "e", "f", "g"},
			[]string{"a", "b", "c", "d", "e", "f", "g"},
		)
		require.NoError(t, err)
		require.Equal(t, 4, lru.Len())
	},
		ExpectedStats{}.WithKeysWritten(7).WithEvictions(3),
	)
}

func TestMSetTooManyTwice(t *testing.T) {
	doShardedTest(t, 2, time.Hour, 2, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet(
			[]string{"a", "b", "c", "d", "e", "f", "g"},
			[]string{"a", "b", "c", "d", "e", "f", "g"},
		)
		require.NoError(t, err)
		require.Equal(t, 4, lru.Len())
		found := lru.MGet("a", "b", "c", "d", "e", "f", "g")
		require.Equal(t, 4, len(found))

		// "g" will still be in the set, but "a" will evict something
		err = lru.MSet(
			[]string{"a", "g"},
			[]string{"a", "g"},
		)

		require.NoError(t, err)
		require.Equal(t, 4, lru.Len())
		_, ok := lru.Get("f")
		require.True(t, ok)
		_, ok = lru.Get("g")
		require.True(t, ok)
	},
		ExpectedStats{}.
			WithKeysWritten(9).
			WithEvictions(4).
			WithKeysReadOK(6).
			WithKeysReadNotFound(3),
	)
}

func TestMGetExpired(t *testing.T) {
	doShardedTest(t, 5, time.Millisecond, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Set("abloy", "medeco")
		time.Sleep(time.Millisecond * 10)

		found := lru.MGet("abloy")
		require.Equal(t, 0, len(found))
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReadExpired(0).
			WithKeysReadNotFound(1).
			WithKeysReaped(1),
	)
}

func TestClose(t *testing.T) {
	lru := sharded.NewT[string, string](10, time.Hour, 10, sharded.StringSharder)
	require.True(t, lru.IsRunning())
	lru.Close()
	time.Sleep(time.Millisecond * 10)
	require.False(t, lru.IsRunning())
	lru.Close() // ensure double-close is safe
}

func TestCloseWithReap(t *testing.T) {
	doShardedTest(t, 10, 10*time.Millisecond, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, int]) {
		require.True(t, lru.IsRunning())

		lru.SetTTL("abloy", 0, time.Hour)
		err := lru.MSetTTL(
			[]string{"a", "b", "c", "d", "e"},
			[]int{1, 2, 3, 4, 5},
			1,
		)
		require.NoError(t, err)
		require.Equal(t, 6, lru.Len())
		time.Sleep(time.Millisecond * 20)
		require.True(t, lru.IsRunning())
		require.Equal(t, 1, lru.Len())
		lru.Close()
		time.Sleep(time.Millisecond * 10)
		require.False(t, lru.IsRunning())
	},
		ExpectedStats{}.
			WithKeysWritten(6).
			WithKeysReaped(5),
	)
}

func TestReap(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Reap()

		lru.SetTTL("abloy", "medeco", time.Millisecond*10)

		found := lru.MGet("abloy")
		require.Equal(t, 1, len(found))

		time.Sleep(time.Millisecond * 10)
		lru.Reap()
		found = lru.MGet("abloy")
		require.Equal(t, 0, len(found))
		require.Equal(t, 0, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReadOK(1).
			WithKeysReadNotFound(1).
			WithKeysReaped(1).
			// make sure that we actually reaped the key, not that the read of an
			// expired key did it
			WithKeysReadExpired(0),
	)
}

func TestPushBeyondCapacity(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		keys := make([]string, 1000)
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			lru.Set(keys[i], keys[i])
		}

		cnt := 0
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			if v, ok := lru.Get(keys[i]); ok {
				cnt++
				require.Equal(t, keys[i], v)
			}
		}
		require.Equal(t, 100, cnt)
	},
		ExpectedStats{}.
			WithKeysWritten(1000).
			WithKeysReadOK(100).
			WithKeysReadNotFound(900).
			WithEvictions(900),
	)
}

func TestPushBeyondCapacitySave28(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		keys := make([]string, 1000)
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			lru.Set(keys[i], keys[i])
			if i >= 28 {
				_, ok := lru.Get("28") // keep 28 hot
				require.True(t, ok, "failed on cycle %d", i)
				if !ok {
					break
				}
			}
		}
		_, ok28 := lru.Get("28")
		require.True(t, ok28, "28")
		_, ok27 := lru.Get("27")
		require.False(t, ok27, "27")
	},
		ExpectedStats{}.
			WithKeysWritten(1000).
			WithKeysReadOK(1000+1-28).
			WithKeysReadNotFound(1),
	)
}

func TestPushBeyondCapacitySave28WithMGet(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		keys := make([]string, 1000)
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			lru.Set(keys[i], keys[i])
			if i >= 28 {
				d := lru.MGet("28") // keep 28 hot
				_, ok := d["28"]
				require.True(t, ok, "failed on cycle %d", i)
				if !ok {
					break
				}
			}
		}
		_, ok28 := lru.Get("28")
		require.True(t, ok28, "28")
		_, ok27 := lru.Get("27")
		require.False(t, ok27, "27")
	},
		ExpectedStats{}.
			WithKeysWritten(1000).
			WithKeysReadOK(1000+1-28).
			WithKeysReadNotFound(1),
	)
}

func TestGetExpired(t *testing.T) {
	doShardedTest(t, 10, 0, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Set("a", "a")
		require.Equal(t, 1, lru.Len())
		v, ok := lru.Get("a")
		require.False(t, ok)
		require.Equal(t, "", v)
		require.Equal(t, 0, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReadExpired(1),
	)
}

func TestExpireCleanup(t *testing.T) {
	doShardedTest(t, 10, 1, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Set("a", "a")
		require.Equal(t, 1, lru.Len())
		time.Sleep(time.Millisecond * 100)
		require.Equal(t, 0, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReaped(1),
	)
}

func TestMGetSomeExpired(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		lru.Set("a", "a")
		lru.SetTTL("b", "b", 0)
		require.Equal(t, 2, lru.Len())
		vals := lru.MGet("a", "b")
		require.Equal(t, 1, len(vals))
		v, ok := vals["a"]
		require.True(t, ok)
		require.Equal(t, "a", v)
		require.Equal(t, 1, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(2).
			WithKeysReadOK(1).
			WithKeysReadExpired(1),
	)
}

func TestMSetOneItem(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet([]string{"a"}, []string{"a"})
		require.NoError(t, err)
		require.Equal(t, 1, lru.Len())
		vals := lru.MGet("a", "b")
		require.Equal(t, 1, len(vals))
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReadOK(1).
			WithKeysReadNotFound(1),
	)
}

func TestMGetEmpty(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet(nil, nil)
		require.NoError(t, err)
		require.Equal(t, 0, lru.Len())
		vals := lru.MGet("a", "b")
		require.Equal(t, 0, len(vals))
	},
		ExpectedStats{}.
			WithKeysWritten(0).
			WithKeysReadOK(0).
			WithKeysReadNotFound(2),
	)
}

func TestMSetEmpty(t *testing.T) {
	doShardedTest(t, 10, time.Hour, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		vals := lru.MGet()
		require.Equal(t, 0, len(vals))
	},
		ExpectedStats{}.
			WithKeysWritten(0).
			WithKeysReadOK(0).
			WithKeysReadNotFound(0),
	)
}

func TestMSetZeroTTL(t *testing.T) {
	doShardedTest(t, 10, 0, 10, sharded.StringSharder, func(t *testing.T, lru *sharded.LazyLRU[string, string]) {
		err := lru.MSet([]string{"a"}, []string{"a"})
		require.NoError(t, err)
		require.Equal(t, 0, lru.Len())
		vals := lru.MGet("a", "b")
		require.Equal(t, 0, len(vals))
	},
		ExpectedStats{}.
			WithKeysWritten(0).
			WithKeysReadOK(0).
			WithKeysReadNotFound(2),
	)
}
