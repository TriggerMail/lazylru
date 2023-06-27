package sharded

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyShardHelper(t *testing.T) {
	ks := newKeyShardHelper(
		[]string{"a", "b", "c"},
		func(string) int {
			return 123
		})
	require.Equal(t, 123, ks.shardIndices[0])
	require.Equal(t, 123, ks.shardIndices[1])
	require.Equal(t, 123, ks.shardIndices[2])
	ix, keys := ks.TakeGroup()
	require.Equal(t, 123, ix)
	require.Equal(t, 3, len(keys))
	ix, keys = ks.TakeGroup()
	require.Equal(t, -1, ix)
	require.Equal(t, 0, len(keys))

	ks = &keyShardHelper[string]{
		[]string{"a", "b", "c"},
		[]int{0, 1, 1},
	}
	ix, keys = ks.TakeGroup()
	require.Equal(t, 0, ix)
	require.Equal(t, 1, len(keys))
	ix, keys = ks.TakeGroup()
	require.Equal(t, 1, ix)
	require.Equal(t, 2, len(keys))
	ix, keys = ks.TakeGroup()
	require.Equal(t, -1, ix)
	require.Equal(t, 0, len(keys))
}

func TestKVShardHelper(t *testing.T) {
	ks := newKVShardHelper(
		[]string{"ak", "bk", "ck"},
		[]string{"av", "bv", "cv"},
		func(k string) int {
			return int(StringSharder(k) % 10)
		},
	)
	require.Equal(t, 3, len(ks.keys))
	for i := 0; i < len(ks.keys); i++ {
		require.Equal(t, int(StringSharder(ks.keys[i])%10), ks.shardIndices[i])
		// compare first letters for match
		require.Equal(t, ks.keys[i][0], ks.values[i][0])
	}
	ix, keys, vals := ks.TakeGroup()
	for ix > 0 {
		for i := 0; i < len(keys); i++ {
			require.Equal(t, ix, int(StringSharder(keys[i])%10))
			require.Equal(t, keys[i][0], vals[i][0])
		}
		ix, keys, vals = ks.TakeGroup()
	}
	require.Equal(t, -1, ix)
	require.Equal(t, 0, len(keys))
	require.Equal(t, 0, len(vals))
	require.Equal(t, 0, len(ks.keys))
	require.Equal(t, 0, len(ks.values))

	ks = &kvShardHelper[string, string]{
		[]string{"ak", "bk", "ck"},
		[]string{"av", "bv", "cv"},
		[]int{0, 1, 1},
	}
	ix, keys, vals = ks.TakeGroup()
	require.Equal(t, 0, ix)
	require.Equal(t, 1, len(keys))
	require.Equal(t, 1, len(vals))
	ix, keys, vals = ks.TakeGroup()
	require.Equal(t, 1, ix)
	require.Equal(t, 2, len(keys))
	require.Equal(t, 2, len(vals))
	ix, keys, vals = ks.TakeGroup()
	require.Equal(t, -1, ix)
	require.Equal(t, 0, len(keys))
	require.Equal(t, 0, len(vals))
}
