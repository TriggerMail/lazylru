package lazylru_test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/TriggerMail/lazylru"
)

func benchmarker(b *testing.B, capacity int, keyCount int, readRate float64) {
	lru := lazylru.New(capacity, time.Minute)
	defer lru.Close()
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = strconv.FormatInt(int64(i), 10)
		lru.Set(keys[i], keys[i])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ix := rand.Intn(len(keys))
		if rand.Float64() < readRate {
			lru.Get(keys[ix])
		} else {
			lru.Set(keys[ix], keys[ix])
		}
	}
}

func Benchmark100Write0ReadOversize(b *testing.B) {
	benchmarker(b, 1000, 100, 0.0)
}

func Benchmark0Write1000ReadOversize(b *testing.B) {
	benchmarker(b, 1000, 100, 1.0)
}

func Benchmark75Write25ReadOversize(b *testing.B) {
	benchmarker(b, 1000, 100, 0.25)
}

func Benchmark25Write75ReadOversize(b *testing.B) {
	benchmarker(b, 1000, 100, 0.75)
}

func Benchmark01Write99ReadOversize(b *testing.B) {
	benchmarker(b, 1000, 100, 0.99)
}

func Benchmark100Write0ReadUnderSize(b *testing.B) {
	benchmarker(b, 100, 1000, 0.0)
}

func Benchmark0Write1000ReadUnderSize(b *testing.B) {
	benchmarker(b, 100, 1000, 1.0)
}

func Benchmark75Write25ReadUnderSize(b *testing.B) {
	benchmarker(b, 100, 1000, 0.25)
}

func Benchmark25Write75ReadUnderSize(b *testing.B) {
	benchmarker(b, 100, 1000, 0.75)
}

func Benchmark01Write99ReadUnderSize(b *testing.B) {
	benchmarker(b, 100, 1000, 0.99)
}

func Benchmark100Write0ReadExactSize(b *testing.B) {
	benchmarker(b, 100, 100, 0.0)
}

func Benchmark0Write1000ReadExactSize(b *testing.B) {
	benchmarker(b, 100, 100, 1.0)
}

func Benchmark75Write25ReadExactSize(b *testing.B) {
	benchmarker(b, 100, 100, 0.25)
}

func Benchmark25Write75ReadExactSize(b *testing.B) {
	benchmarker(b, 100, 100, 0.75)
}

func Benchmark01Write99ReadExactSize(b *testing.B) {
	benchmarker(b, 100, 100, 0.99)
}
