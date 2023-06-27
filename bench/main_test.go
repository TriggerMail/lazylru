package main_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/TriggerMail/lazylru"
	bench "github.com/TriggerMail/lazylru/bench"
	"github.com/stretchr/testify/require"
)

func TestWrapperFunctions(t *testing.T) {
	for _, test := range []struct {
		cache bench.Cache
		name  string
	}{
		{bench.NewMapCache[string, string](10, time.Hour), "mapcache"},
		{lazylru.NewT[string, string](10, time.Hour), "lazylru"},
		{bench.NewHashicorpWrapper[string, string](10), "hashicorp.lru"},
		{bench.NewHashicorpWrapperExp[string, string](10, time.Hour), "hashicorp.exp"},
		{bench.NewHashicorpARCWrapper[string, string](10), "hashicorp.arc"},
		{bench.NewHashicorp2QWrapper[string, string](10), "hashicorp.2Q"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, ok := test.cache.Get("medeco")
			require.False(t, ok)
			test.cache.Set("medeco", "abloy")
			v, ok := test.cache.Get("medeco")
			require.True(t, ok)
			require.Equal(t, "abloy", v)
			_, ok = test.cache.Get("schlage")
			require.False(t, ok)
			test.cache.Close()
		})
	}
	// the NullCache doesn't actually hold anything, so we need a different test
	t.Run("NullCache", func(t *testing.T) {
		cache := bench.NullCache
		_, ok := cache.Get("medeco")
		require.False(t, ok)
		cache.Set("medeco", "abloy")
		_, ok = cache.Get("medeco")
		require.False(t, ok)
		_, ok = cache.Get("schlage")
		require.False(t, ok)
		cache.Close()
	})
}

func TestRoundDigits(t *testing.T) {
	for _, test := range []struct {
		in   float64
		prec int
		exp  float64
	}{
		{1, 0, 1},
		{1, 1, 1},
		{1, 10, 1},
		{1.4, 0, 1.0},
		{1.6, 0, 2.0},
		{1.449, 1, 1.4},
		{1.44449, 3, 1.444},
		{1.45, 1, 1.5},
	} {
		t.Run(fmt.Sprintf("%f@%d", test.in, test.prec), func(t *testing.T) {
			act := bench.RoundDigits(test.in, test.prec)
			require.Equal(t, test.exp, act)
		})
	}
}

func TestSpinsPerMicro(t *testing.T) {
	require.Less(t, uint64(10), bench.SpinsPerMicro)
}

func TestSourceData(t *testing.T) {
	r := bench.NewTestData(1000)
	require.Equal(t, 1000, len(r))
	for i := 1; i < len(r); i++ {
		require.NotEqual(t, r[i-1], r[i])
	}
}
