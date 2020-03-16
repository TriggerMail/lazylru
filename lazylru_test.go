package lazylru_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/TriggerMail/lazylru"
	"github.com/stretchr/testify/assert"
)

func doTest(t *testing.T, maxItems int, ttl time.Duration, test func(t *testing.T, lru *lazylru.LazyLRU), expected ExpectedStats) {
	lru := lazylru.New(maxItems, ttl)
	test(t, lru)
	lru.Close()
	expected.Test(t, lru.Stats())
}

func TestMakeNew(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		assert.NotNil(t, lru)
	},
		ExpectedStats{},
	)
}

func TestGetUnknown(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		v, ok := lru.Get("something new")
		assert.Nil(t, v)
		assert.False(t, ok)
	},
		ExpectedStats{}.WithKeysReadNotFound(1),
	)
}

func TestGetKnown(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Set("abloy", "medeco")
		v, ok := lru.Get("abloy")
		assert.True(t, ok)
		vstr, vok := v.(string)
		assert.True(t, vok)
		assert.Equal(t, "medeco", vstr)
	},
		ExpectedStats{}.WithKeysWritten(1).WithKeysReadOK(1),
	)
}

func testGetKnownShuffleMitigationHelper(t *testing.T, getter func(lru *lazylru.LazyLRU, key string) (interface{}, bool)) {
	doTest(t, 100, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		keys := make([]string, 100)
		values := make([]interface{}, len(keys))
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			values[i] = i
		}
		assert.NoError(t, lru.MSet(keys, values))

		// This should affect 100 reads, but only 1 shuffle
		for i := 0; i < 100; i++ {
			v, ok := getter(lru, "0")
			assert.True(t, ok)
			vint, vok := v.(int)
			assert.True(t, vok)
			assert.Equal(t, 0, vint)
		}

		// This should affect 100 reads, but only no shuffles
		for i := 0; i < 100; i++ {
			v, ok := getter(lru, "99")
			assert.True(t, ok)
			vint, vok := v.(int)
			assert.True(t, vok)
			assert.Equal(t, 99, vint)
		}
	},
		ExpectedStats{}.WithKeysReadOK(200).WithShuffles(1),
	)
}

func TestGetKnownShuffleMitigationGet(t *testing.T) {
	testGetKnownShuffleMitigationHelper(t,
		func(lru *lazylru.LazyLRU, key string) (interface{}, bool) {
			v, ok := lru.Get(key)
			return v, ok
		})
}
func TestGetKnownShuffleMitigationMGet(t *testing.T) {
	testGetKnownShuffleMitigationHelper(t,
		func(lru *lazylru.LazyLRU, key string) (interface{}, bool) {
			d := lru.MGet(key)
			v, ok := d[key]
			return v, ok
		},
	)
}

func TestMGetUnknown(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		found := lru.MGet("a", "b", "c")
		assert.Equal(t, 0, len(found))
	},
		ExpectedStats{}.WithKeysReadNotFound(3),
	)
}

func TestMGetKnown(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		err := lru.MSet(
			[]string{"abloy", "schlage"},
			[]interface{}{"medeco", "kwikset"},
		)
		assert.NoError(t, err)
		found := lru.MGet("abloy", "schlage")
		assert.Equal(t, 2, len(found))
		v, ok := found["abloy"]
		assert.True(t, ok)
		vstr, vok := v.(string)
		assert.True(t, vok)
		assert.Equal(t, "medeco", vstr)
	},
		ExpectedStats{}.WithKeysWritten(2).WithKeysReadOK(2),
	)
}

func TestSetNTimes(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		assert.Equal(t, 0, lru.Len())
		lru.Set("abloy", "schlage")
		assert.Equal(t, 1, lru.Len())
		for i := 0; i < 1000; i++ {
			lru.Set("abloy", "schlage")
		}
		assert.Equal(t, 1, lru.Len())
	},
		ExpectedStats{}.WithKeysWritten(1001),
	)
}

func TestMGetOneKnown(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Set("abloy", "medeco")

		found := lru.MGet("abloy")
		assert.Equal(t, 1, len(found))

		v, ok := found["abloy"]
		assert.True(t, ok)
		vstr, vok := v.(string)
		assert.True(t, vok)
		assert.Equal(t, "medeco", vstr)
	},
		ExpectedStats{}.WithKeysWritten(1).WithKeysReadOK(1),
	)
}

func TestMSetBad(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		err := lru.MSet(
			[]string{"abloy"},
			[]interface{}{"medeco", "kwikset"},
		)
		assert.Error(t, err)
	},
		ExpectedStats{}.WithKeysWritten(0),
	)
}

func TestMSetTooMany(t *testing.T) {
	doTest(t, 5, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		err := lru.MSet(
			[]string{"a", "b", "c", "d", "e", "f", "g"},
			[]interface{}{"a", "b", "c", "d", "e", "f", "g"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 5, lru.Len())
	},
		ExpectedStats{}.WithKeysWritten(7).WithEvictions(2),
	)
}

func TestMSetTooManyTwice(t *testing.T) {
	doTest(t, 5, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		err := lru.MSet(
			[]string{"a", "b", "c", "d", "e", "f", "g"},
			[]interface{}{"a", "b", "c", "d", "e", "f", "g"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 5, lru.Len())
		found := lru.MGet("a", "b", "c", "d", "e", "f", "g")
		assert.Equal(t, 5, len(found))

		// "g" will still be in the set, but "a" will evict something
		err = lru.MSet(
			[]string{"a", "g"},
			[]interface{}{"a", "g"},
		)

		assert.NoError(t, err)
		assert.Equal(t, 5, lru.Len())
		_, ok := lru.Get("f")
		assert.True(t, ok)
		_, ok = lru.Get("g")
		assert.True(t, ok)
	},
		ExpectedStats{}.
			WithKeysWritten(9).
			WithEvictions(3).
			WithKeysReadOK(7).
			WithKeysReadNotFound(2),
	)
}

func TestMGetExpired(t *testing.T) {
	doTest(t, 5, time.Millisecond, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Set("abloy", "medeco")
		time.Sleep(time.Millisecond * 10)

		found := lru.MGet("abloy")
		assert.Equal(t, 0, len(found))
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReadExpired(0).
			WithKeysReadNotFound(1).
			WithKeysReaped(1),
	)
}

func TestClose(t *testing.T) {
	lru := lazylru.New(10, time.Hour)
	assert.True(t, lru.IsRunning())
	lru.Close()
	time.Sleep(time.Millisecond * 10)
	assert.False(t, lru.IsRunning())
	lru.Close() // ensure double-close is safe
}

func TestCloseWithReap(t *testing.T) {
	doTest(t, 10, 10*time.Millisecond, func(t *testing.T, lru *lazylru.LazyLRU) {
		assert.True(t, lru.IsRunning())

		lru.SetTTL("abloy", "medeco", time.Hour)
		err := lru.MSetTTL(
			[]string{"a", "b", "c", "d", "e"},
			[]interface{}{1, 2, 3, 4, 5},
			0,
		)
		assert.NoError(t, err)
		assert.Equal(t, 6, lru.Len())
		time.Sleep(time.Millisecond * 20)
		assert.True(t, lru.IsRunning())
		assert.Equal(t, 1, lru.Len())
		lru.Close()
		time.Sleep(time.Millisecond * 10)
		assert.False(t, lru.IsRunning())
	},
		ExpectedStats{}.
			WithKeysWritten(6).
			WithKeysReaped(5),
	)
}

func TestReap(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Reap()

		lru.SetTTL("abloy", "medeco", time.Millisecond*10)

		found := lru.MGet("abloy")
		assert.Equal(t, 1, len(found))

		time.Sleep(time.Millisecond * 10)
		lru.Reap()
		found = lru.MGet("abloy")
		assert.Equal(t, 0, len(found))
		assert.Equal(t, 0, lru.Len())
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
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		keys := make([]string, 100)
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			lru.Set(keys[i], keys[i])
		}

		for _, key := range keys[:90] {
			_, ok := lru.Get(key)
			assert.False(t, ok, "key: %s", key)
		}
		for _, key := range keys[90:] {
			v, ok := lru.Get(key)
			assert.True(t, ok, "key: %s", key)
			vstr, vok := v.(string)
			assert.True(t, vok, "key: %s", key)
			assert.Equal(t, key, vstr, "key: %s", key)
		}
	},
		ExpectedStats{}.
			WithKeysWritten(100).
			WithKeysReadOK(10).
			WithKeysReadNotFound(90).
			WithEvictions(90),
	)
}

func TestPushBeyondCapacitySave28(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		keys := make([]string, 100)
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			lru.Set(keys[i], keys[i])
			if i >= 28 {
				_, ok := lru.Get("28") // keep 28 hot
				assert.True(t, ok, "failed on cycle %d", i)
				if !ok {
					break
				}
			}
		}
		_, ok28 := lru.Get("28")
		assert.True(t, ok28, "28")
		_, ok27 := lru.Get("27")
		assert.False(t, ok27, "27")
	},
		ExpectedStats{}.
			WithKeysWritten(100).
			WithKeysReadOK(100+1-28).
			WithKeysReadNotFound(1),
	)
}
func TestPushBeyondCapacitySave28WithMGet(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		keys := make([]string, 100)
		for i := 0; i < len(keys); i++ {
			keys[i] = strconv.FormatInt(int64(i), 10)
			lru.Set(keys[i], keys[i])
			if i >= 28 {
				d := lru.MGet("28") // keep 28 hot
				_, ok := d["28"]
				assert.True(t, ok, "failed on cycle %d", i)
				if !ok {
					break
				}
			}
		}
		_, ok28 := lru.Get("28")
		assert.True(t, ok28, "28")
		_, ok27 := lru.Get("27")
		assert.False(t, ok27, "27")
	},
		ExpectedStats{}.
			WithKeysWritten(100).
			WithKeysReadOK(100+1-28).
			WithKeysReadNotFound(1),
	)
}

func TestGetExpired(t *testing.T) {
	doTest(t, 10, 0, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Set("a", "a")
		assert.Equal(t, 1, lru.Len())
		v, ok := lru.Get("a")
		assert.False(t, ok)
		assert.Nil(t, v)
		assert.Equal(t, 0, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReadExpired(1),
	)
}

func TestExpireCleanup(t *testing.T) {
	doTest(t, 10, 1, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Set("a", "a")
		assert.Equal(t, 1, lru.Len())
		time.Sleep(time.Millisecond * 100)
		assert.Equal(t, 0, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(1).
			WithKeysReaped(1),
	)
}

func TestMGetSomeExpired(t *testing.T) {
	doTest(t, 10, time.Hour, func(t *testing.T, lru *lazylru.LazyLRU) {
		lru.Set("a", "a")
		lru.SetTTL("b", "b", 0)
		assert.Equal(t, 2, lru.Len())
		vals := lru.MGet("a", "b")
		assert.Equal(t, 1, len(vals))
		v, ok := vals["a"]
		assert.True(t, ok)
		assert.Equal(t, "a", v.(string))
		assert.Equal(t, 1, lru.Len())
	},
		ExpectedStats{}.
			WithKeysWritten(2).
			WithKeysReadOK(1).
			WithKeysReadExpired(1),
	)
}
