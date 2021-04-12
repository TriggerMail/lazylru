package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/TriggerMail/lazylru"
	"go.uber.org/zap"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Close()
}

type TestParams struct {
	Duration time.Duration
	Threads  int
	Size     int
	Name     string
	Cache    Cache
}

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()

	logger.Info("Begin")
	testData := NewTestDataRanges(5, 1000, 1000000)
	logger.Info("Data loaded")
	testDuration := 5 * time.Second

	caches := []struct {
		name    string
		factory func(int) Cache
	}{
		{"null", func(size int) Cache { return NullCache }},
		{"lazylru.hour", func(size int) Cache { return lazylru.New(size, time.Hour) }},
		{"lazylru.50ms", func(size int) Cache { return lazylru.New(size, time.Millisecond*50) }},
		{"hashicorp.lru", func(size int) Cache { return NewHashicorpWrapper(size) }},
		{"hashicorp.arc", func(size int) Cache { return NewHashicorpARCWrapper(size) }},
		{"hashicorp.2Q", func(size int) Cache { return NewHashicorp2QWrapper(size) }},
	}

	for _, testThreads := range []int{1, 4} {
		for _, testSize := range []int{10, 1000, 10000} {
			for _, cache := range caches {
				testLru(TestParams{testDuration, testThreads, testSize, cache.name, cache.factory(testSize)}, testData)
				_ = logger.Sync()
			}
		}
	}
}

func testLru(testParams TestParams, testData TestData) {
	runtime := testParams.Duration
	threads := testParams.Threads
	cache := testParams.Cache
	log := logger.With(zap.String("name", testParams.Name), zap.Int("size", testParams.Size), zap.Int("threads", threads))

	var wg sync.WaitGroup
	globalHits := int64(0)
	globalCycles := int64(0)

	N := int64(1<<63 - 1)

	endtimes := time.NewTimer(runtime)
	go func() {
		<-endtimes.C
		log.Debug("Signalling the end times")
		N = -1
	}()

	log.Debug("Starting threads.", zap.Int("count", threads))
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			log.Debug("Starting thread", zap.Int("thread", i))
			hits := int64(0)
			cycles := int64(0)
			for ; cycles < N; cycles++ {
				if cycles%10000000 == 0 {
					log.Debug(
						"Progress",
						zap.Int("thread", i),
						zap.Int64("count", cycles),
						zap.Int64("N", N),
					)
				}
				key, value := testData.RandomKV()
				if _, ok := cache.Get(key); ok {
					hits++
				} else {
					cache.Set(key, value)
				}
			}
			atomic.AddInt64(&globalHits, hits)
			atomic.AddInt64(&globalCycles, int64(cycles))
			log.Debug("Stopping thread", zap.Int("thread", i))
		}(i)
	}
	log.Debug("Waiting for threads to finish")
	wg.Wait()
	log.Debug("All threads finished. Closing lru")
	cache.Close()
	// stats := lru.Stats()
	// log.Info(
	// 	"stats",
	// 	zap.Uint32("keys_written", stats.KeysWritten),
	// 	zap.Uint32("read_ok", stats.KeysReadOK),
	// 	zap.Uint32("read_not_found", stats.KeysReadNotFound),
	// 	zap.Uint32("read_expired", stats.KeysReadExpired),
	// 	zap.Uint32("shuffles", stats.Shuffles),
	// 	zap.Uint32("evictions", stats.Evictions),
	// 	zap.Uint32("reaped", stats.KeysReaped),
	// 	zap.Uint32("reaper_cycles", stats.ReaperCycles),
	// )
	log.Info(
		"Done",
		zap.Int64("cycles", globalCycles),
		zap.Duration("runtime", runtime),
		zap.Float64("rate_kHz", float64(globalCycles)/(runtime.Seconds()*1000)),
		zap.Float64("hit_rate", float64(globalHits)*100.0/float64(globalCycles)),
	)
}
