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
	lazylruT "github.com/TriggerMail/lazylru/generic"
	"go.uber.org/zap"
)

// SpinsPerMicro is a rough, empirical measure of the number of cycles the
// spinWait function must run per microsecond
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

// Cache is the interface that all implementations under test must implement
type Cache interface {
	Get(key string) (string, bool)
	Set(key string, value string)
	Close()
}

// TestParams holds the parameters of a test
type TestParams struct {
	Duration     time.Duration
	MaxCycles    int
	Threads      int
	Size         int
	Name         string
	Cache        Cache
	WorkTime     time.Duration
	SleepTime    time.Duration
	TestDataSpec TestDataSpec
}

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction(zap.WithCaller(false))

	logger.Info("Begin")
	logger.Info("Data loaded")
	testDuration := 5 * time.Second

	caches := []struct {
		name    string
		factory func(int) Cache
	}{
		{"null", func(size int) Cache { return NullCache }},
		{"mapcache.hour", func(size int) Cache { return NewMapCache[string, string](size, time.Hour) }},
		{"mapcache.50ms", func(size int) Cache { return NewMapCache[string, string](size, time.Millisecond*50) }},
		{"lazylru.hour", func(size int) Cache { return (*LazyLRUTypesafe[string])(lazylru.New(size, time.Hour)) }},
		{"lazylru.50ms", func(size int) Cache { return (*LazyLRUTypesafe[string])(lazylru.New(size, time.Millisecond*50)) }},
		{"lazylruT.hour", func(size int) Cache { return lazylruT.NewT[string, string](size, time.Hour) }},
		{"lazylruT.50ms", func(size int) Cache { return lazylruT.NewT[string, string](size, time.Millisecond*50) }},
		{"hashicorp.lru", func(size int) Cache { return NewHashicorpWrapper[string, string](size) }},
		{"hashicorp.exp_hour", func(size int) Cache { return NewHashicorpWrapperExp[string, string](size, time.Hour) }},
		{"hashicorp.exp_50ms", func(size int) Cache { return NewHashicorpWrapperExp[string, string](size, time.Millisecond*50) }},
		{"hashicorp.arc", func(size int) Cache { return NewHashicorpARCWrapper[string, string](size) }},
		{"hashicorp.2Q", func(size int) Cache { return NewHashicorp2QWrapper[string, string](size) }},
	}

	printHeaders()

	for _, tds := range []TestDataSpec{
		{1, 5000, 1000000},
		{5, 1000, 1000000},
		{5, 20000, 1000000},
	} {
		testData := tds.ToRanges()
		for _, testSleepTime := range []time.Duration{0, 100 * time.Microsecond, time.Millisecond} {
			for _, testWorkTime := range []time.Duration{0, 1 * time.Microsecond, 10 * time.Microsecond} {
				for _, testThreads := range []int{1, 8, 64, 256} {
					for _, testSize := range []int{100, 10000} {
						for _, cache := range caches {
							testLru(
								TestParams{
									Duration:     testDuration,
									MaxCycles:    tds.Ranges * tds.CyclesPerRange,
									Threads:      testThreads,
									Size:         testSize,
									Name:         cache.name,
									Cache:        cache.factory(testSize),
									WorkTime:     testWorkTime,
									SleepTime:    testSleepTime,
									TestDataSpec: tds,
								},
								testData,
							)
							_ = logger.Sync()
						}
					}
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
		atomic.StoreInt64(&N, -1)
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
			for ; cycles < atomic.LoadInt64(&N); cycles++ {
				if cycles%10000000 == 0 {
					log.Debug(
						"Progress",
						zap.Int("thread", i),
						zap.Int64("count", cycles),
						zap.Int64("N", atomic.LoadInt64(&N)),
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
				time.Sleep(testParams.SleepTime)
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

	// disabled since it only works on lazylru and pollutes the output
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
	printResult(globalCycles, globalHits, duration, testParams)
}

func printHeaders() {
	fmt.Println(strings.Join([]string{
		"algorithm",
		"ranges",
		"keys/range",
		"cycles/range",
		"threads",
		"size",
		"work_time_µs",
		"sleep_time_µs",
		"cycles",
		"duration_ms",
		"rate_kHz",
		"hit_rate_%",
	}, "\t"))
}

func printResult(cycles int64, hits int64, duration time.Duration, testParams TestParams) {
	fmt.Print(testParams.Name)
	fmt.Print("\t")
	fmt.Print(testParams.TestDataSpec.Ranges)
	fmt.Print("\t")
	fmt.Print(testParams.TestDataSpec.KeysPerRange)
	fmt.Print("\t")
	fmt.Print(testParams.TestDataSpec.CyclesPerRange)
	fmt.Print("\t")
	fmt.Print(testParams.Threads)
	fmt.Print("\t")
	fmt.Print(testParams.Size)
	fmt.Print("\t")
	fmt.Print(testParams.WorkTime / time.Microsecond)
	fmt.Print("\t")
	fmt.Print(testParams.SleepTime / time.Microsecond)
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

// RoundDigits rounds a floating point value to a given number of digits
func RoundDigits(val float64, digits int) float64 {
	scale := math.Pow10(digits)
	return math.Round(val*scale) / scale
}
