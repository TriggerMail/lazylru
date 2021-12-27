package sharded_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	lazylru "github.com/TriggerMail/lazylru/generic"
	sharded "github.com/TriggerMail/lazylru/generic/sharded"
)

const keycnt = 1 << 14

var keys = func() []string {
	k := make([]string, keycnt)

	for i := 0; i < keycnt; i++ {
		k[i] = strconv.Itoa(i)
	}
	return k
}()

type benchconfig struct {
	capacity    int
	keyCount    int
	readRate    float64
	shards      int
	concurrency int
}

func (bc benchconfig) Name() string {
	comment := "eqcap"
	if bc.capacity > bc.keyCount {
		comment = "overcap"
	} else if bc.capacity < bc.keyCount {
		comment = "undercap"
	}
	return fmt.Sprintf("%dW/%dR/%dS/%dT_%s", 100-int(100*bc.readRate), int(100*bc.readRate), bc.shards, bc.concurrency, comment)
}

func (bc benchconfig) Run(b *testing.B) {
	var lru interface {
		Set(string, int)
		Get(string) (int, bool)
		Close()
	}

	if bc.shards <= 1 {
		lru = lazylru.NewT[string, int](bc.capacity, time.Minute)
	} else {
		if bc.capacity < bc.shards {
			b.Fatalf("Capacity (%d) must be greater than shard count (%d)", bc.capacity, bc.shards)
		}
		lru = sharded.NewT[string, int](bc.capacity/bc.shards, time.Minute, bc.shards, sharded.StringSharder)
	}

	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], i)
	}
	runtime.GC()
	b.ResetTimer()
	var wg sync.WaitGroup
	wg.Add(bc.concurrency)
	baseTime := time.Now().UnixNano()

	for c := 0; c < bc.concurrency; c++ {
		go func(c int) {
			rnd := rand.New(rand.NewSource(baseTime + int64(c))) //nolint:gosec
			for i := c; i < b.N; i += bc.concurrency {
				ix := rnd.Intn(bc.keyCount)
				if rnd.Float64() < bc.readRate {
					lru.Get(keys[ix])
				} else {
					lru.Set(keys[ix], ix)
				}
			}
			wg.Done()
		}(c)
	}
	wg.Wait()
	b.StopTimer()
}

func Benchmark(b *testing.B) {
	for _, cfg := range []struct {
		capacity int
		keyCount int
	}{
		{1 << 8, 1 << 8}, // this is meant as a warm-up
		{1 << 14, 1 << 8},
		{1 << 8, 1 << 14},
		{1 << 14, 1 << 14},
	} {
		if cfg.keyCount > keycnt {
			b.Fatalf("configured keyCount (%d) cannot be greater than the global keycnt (%d)", cfg.keyCount, keycnt)
		}
		for _, readRate := range []float64{0, 0.25, 0.75, 0.99, 1} {
			for _, shards := range []int{1, 4, 16, 64} {
				for _, concurrency := range []int{1, 1 << 4, 1 << 8, 1 << 16} {
					bc := benchconfig{
						capacity:    cfg.capacity,
						keyCount:    cfg.keyCount,
						readRate:    readRate,
						shards:      shards,
						concurrency: concurrency,
					}
					b.Run(bc.Name(), bc.Run)
				}
			}
		}
	}
}
