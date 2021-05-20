package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TriggerMail/lazylru"
	"go.uber.org/zap"
)

var SpinsPerMicro = func() uint64 {
	testSpins := uint64(1000000)
	testIterations := 1001
	results := make([]time.Duration, testIterations)

	for i := 0; i < testIterations; i++ {
		start := time.Now()
		spinWait(testSpins)
		results[i] = time.Since(start)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	return uint64(float64(testSpins) / (results[testIterations/2].Seconds() * 1000000))
}()

func spinWait(n uint64) {
	for i := uint64(0); i < n; i++ {
		_ = i
	}
}

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Close()
}

type TestParams struct {
	Duration    time.Duration
	MaxCycles   int
	Threads     int
	Size        int
	Name        string
	Cache       Cache
	WorkTime    time.Duration
	RangeCount  int
	RangeSize   int
	RangeCycles int
}

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction(zap.WithCaller(false))

	logger.Info("Begin")
	rangeCount := 5
	rangeSize := 1000
	rangeCycles := 1000000
	testData := NewTestDataRanges(rangeCount, rangeSize, rangeCycles)
	logger.Info("Data loaded")
	testDuration := 5 * time.Second

	caches := []struct {
		name    string
		factory func(int) Cache
	}{
		{"null", func(size int) Cache { return NullCache }},
		{"mapcache.hour", func(size int) Cache { return NewMapCache(size, time.Hour) }},
		{"mapcache.50ms", func(size int) Cache { return NewMapCache(size, time.Millisecond*50) }},
		{"lazylru.hour", func(size int) Cache { return lazylru.New(size, time.Hour) }},
		{"lazylru.50ms", func(size int) Cache { return lazylru.New(size, time.Millisecond*50) }},
		{"hashicorp.lru", func(size int) Cache { return NewHashicorpWrapper(size) }},
		{"hashicorp.exp_hour", func(size int) Cache { return NewHashicorpWrapperExp(size, time.Hour) }},
		{"hashicorp.exp_50ms", func(size int) Cache { return NewHashicorpWrapperExp(size, time.Millisecond*50) }},
		{"hashicorp.arc", func(size int) Cache { return NewHashicorpARCWrapper(size) }},
		{"hashicorp.2Q", func(size int) Cache { return NewHashicorp2QWrapper(size) }},
	}

	printHeaders()
	// for _, testWorkTime := range []time.Duration{0, time.Microsecond, 10 * time.Microsecond, 100 * time.Microsecond} {
	// 	for _, testThreads := range []int{1, 2, 4, 8, 16} {
	// for _, testSize := range []int{2000, 10000} {
	for _, testWorkTime := range []time.Duration{0, 1 * time.Microsecond} {
		for _, testThreads := range []int{1, 8, 64} {
			for _, testSize := range []int{100, 10000} {
				for _, cache := range caches {
					testLru(
						TestParams{
							Duration:    testDuration,
							MaxCycles:   rangeCount * rangeCycles,
							Threads:     testThreads,
							Size:        testSize,
							Name:        cache.name,
							Cache:       cache.factory(testSize),
							WorkTime:    testWorkTime,
							RangeCount:  rangeCount,
							RangeSize:   rangeSize,
							RangeCycles: rangeCycles,
						},
						testData,
					)
					_ = logger.Sync()
				}
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

	N := int64(testParams.MaxCycles) / int64(threads)
	if N <= 0 {
		N = int64(1<<63 - 1)
	}

	endtimes := time.NewTimer(runtime)
	go func() {
		<-endtimes.C
		log.Debug("Signalling the end times")
		N = -1
	}()

	workCycles := uint64(testParams.WorkTime/time.Microsecond) * SpinsPerMicro

	log.Debug("Starting threads.", zap.Int("count", threads))
	start := time.Now()
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
				if workCycles > 0 {
					spinWait(workCycles)
				}
			}
			atomic.AddInt64(&globalHits, hits)
			atomic.AddInt64(&globalCycles, int64(cycles))
			log.Debug("Stopping thread", zap.Int("thread", i))
		}(i)
	}
	log.Debug("Waiting for threads to finish")
	wg.Wait()
	duration := time.Since(start)
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

	// log.Info(
	// 	"Done",
	// 	zap.Int64("cyc", globalCycles),
	// 	zap.Duration("rt", duration),
	// 	zap.Float64("r_kHz", RoundDigits(float64(globalCycles)/(duration.Seconds()*1000), 2)),
	// 	zap.Float64("hr", RoundDigits(float64(globalHits)*100.0/float64(globalCycles), 2)),
	// 	zap.Uint64("wrk_µs", uint64(testParams.WorkTime/time.Microsecond)),
	// )
	printResult(globalCycles, globalHits, duration, testParams)
}

func printHeaders() {
	fmt.Println(strings.Join([]string{
		"algorithm",
		"ranges",
		"range_size",
		"range_cycles",
		"threads",
		"size",
		"work_time_µs",
		"cycles",
		"duration_ms",
		"rate_kHz",
		"hit_rate_%",
	}, "\t"))
}

func printResult(cycles int64, hits int64, duration time.Duration, testParams TestParams) {
	fmt.Print(testParams.Name)
	fmt.Print("\t")
	fmt.Print(testParams.RangeCount)
	fmt.Print("\t")
	fmt.Print(testParams.RangeSize)
	fmt.Print("\t")
	fmt.Print(testParams.RangeCycles)
	fmt.Print("\t")
	fmt.Print(testParams.Threads)
	fmt.Print("\t")
	fmt.Print(testParams.Size)
	fmt.Print("\t")
	fmt.Print(testParams.WorkTime / time.Microsecond)
	fmt.Print("\t")
	fmt.Print(cycles)
	fmt.Print("\t")
	fmt.Print(int(math.Round(duration.Seconds() * 1000)))
	fmt.Print("\t")
	fmt.Print(RoundDigits(float64(cycles)/(duration.Seconds()*1000), 2))
	fmt.Print("\t")
	fmt.Print(RoundDigits(float64(hits)*100.0/float64(cycles), 2))

	fmt.Println()
}

func RoundDigits(val float64, digits int) float64 {
	scale := math.Pow10(digits)
	return math.Round(val*scale) / scale
}
