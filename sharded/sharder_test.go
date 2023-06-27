package sharded_test

import (
	"math/rand"
	"sort"
	"strconv"
	"testing"

	"github.com/TriggerMail/lazylru/sharded"
	"github.com/stretchr/testify/require"
)

func TestStringSharder(t *testing.T) {
	require.Equal(t,
		sharded.StringSharder("foo"),
		sharded.StringSharder("foo"),
	)

	require.NotEqual(t,
		sharded.StringSharder("foo"),
		sharded.StringSharder("bar"),
	)
}

func TestBytesSharder(t *testing.T) {
	require.Equal(t,
		sharded.BytesSharder([]byte("foo")),
		sharded.BytesSharder([]byte("foo")),
	)

	require.NotEqual(t,
		sharded.BytesSharder([]byte("foo")),
		sharded.BytesSharder([]byte("bar")),
	)
}

func TestStructSharder(t *testing.T) {
	type MyStruct struct {
		name     string
		category string
		count    int
	}

	structSharder := sharded.HashingSharder(func(v MyStruct, h sharded.H) {
		_, err := h.WriteString(v.name)
		require.NoError(t, err)
		_, err = h.WriteString("|")
		require.NoError(t, err)
		_, err = h.WriteString(v.category)
		require.NoError(t, err)
		h.WriteUint64(uint64(v.count))
	})

	testData := []MyStruct{
		{"foo", "cat1", 0},
		{"foo", "cat2", 0},
		{"foo", "cat1", 1},
		{"foo", "cat2", 1},
		{"bar", "cat1", 0},
		{"bar", "cat2", 0},
		{"bar", "cat1", 1},
		{"bar", "cat2", 1},
	}
	shards := make([]int, len(testData))
	for i, td := range testData {
		shards[i] = int(structSharder(td))
	}
	sort.Ints(shards)
	for i := 1; i < len(shards); i++ {
		require.NotEqual(t, shards[i-1], shards[i])
	}
}

func TestStructSharderNums(t *testing.T) {
	type MyStruct struct {
		a int
		b int32
		c int16
		d int8
	}

	structSharder := sharded.HashingSharder(func(v MyStruct, h sharded.H) {
		h.WriteUint64(uint64(v.a))
		h.WriteUint32(uint32(v.b))
		h.WriteUint16(uint16(v.c))
		h.WriteUint8(uint8(v.d))
	})

	testData := []MyStruct{
		{0, 0, 0, 0},
		{0, 0, 0, 1},
		{0, 0, 1, 0},
		{0, 0, 1, 1},
		{0, 1, 0, 0},
		{0, 1, 0, 1},
		{0, 1, 1, 0},
		{0, 1, 1, 1},
		{1, 0, 0, 0},
		{1, 0, 0, 1},
		{1, 0, 1, 0},
		{1, 0, 1, 1},
		{1, 1, 0, 0},
		{1, 1, 0, 1},
		{1, 1, 1, 0},
		{1, 1, 1, 1},
	}
	shards := make([]int, len(testData))
	for i, td := range testData {
		shards[i] = int(structSharder(td))
	}
	sort.Ints(shards)
	for i := 1; i < len(shards); i++ {
		require.NotEqual(t, shards[i-1], shards[i])
	}
}

func BenchmarkStringSharder(b *testing.B) {
	chars := []byte(
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	for _, keysize := range []int{1, 4, 16, 64, 256} {
		b.Run(strconv.Itoa(keysize), func(b *testing.B) {
			sources := make([]string, 1000)
			for i := 0; i < 1000; i++ {
				rand.Shuffle(len(chars), func(i, j int) {
					chars[i], chars[j] = chars[j], chars[i]
				})
				sources[i] = string(chars[0:keysize])
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sharded.StringSharder(sources[i%len(sources)])
			}
		})
	}
}

func BenchmarkBytesSharder(b *testing.B) {
	chars := []byte(
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	for _, keysize := range []int{1, 4, 16, 64, 256} {
		b.Run(strconv.Itoa(keysize), func(b *testing.B) {
			sources := make([][]byte, 1000)
			for i := 0; i < 1000; i++ {
				sources[i] = make([]byte, keysize)
				rand.Shuffle(len(chars), func(i, j int) {
					chars[i], chars[j] = chars[j], chars[i]
				})
				copy(sources[i], chars)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sharded.BytesSharder(sources[i%len(sources)])
			}
		})
	}
}

func BenchmarkCustomSharder(b *testing.B) {
	chars := []byte(
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" +
			"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	type Custom struct {
		a string
		b string
		c int
	}

	sources := make([]Custom, 1000)
	for i := 0; i < 1000; i++ {
		rand.Shuffle(len(chars), func(i, j int) {
			chars[i], chars[j] = chars[j], chars[i]
		})
		sources[i] = Custom{
			a: string(chars[0:12]),
			b: string(chars[0:12]),
			c: i,
		}
	}
	sharder := sharded.HashingSharder(func(k Custom, h sharded.H) {
		_, _ = h.WriteString(k.a)
		_, _ = h.WriteString("|")
		_, _ = h.WriteString(k.b)
		h.WriteUint64(uint64(k.c))
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sharder(sources[i%len(sources)])
	}
}
