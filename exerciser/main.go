package main

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TriggerMail/lazylru"
	"go.uber.org/zap"
)

const validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(size int) string {
	retval := make([]byte, size)
	for i := 0; i < len(retval); i++ {
		retval[i] = validChars[rand.Intn(len(validChars))]
	}
	return string(retval)
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	count := 1000
	runtime := time.Second * 5
	threads := 4
	readRate := 0.9

	logger.Info("Begin")

	lru := lazylru.New(100000, time.Hour)

	keys := make([]string, count)
	values := make([]interface{}, count)

	for i := 0; i < count; i++ {
		keys[i] = randomString(12)
		values[i] = randomString(24)
	}
	if err := lru.MSetTTL(keys, values, time.Hour); err != nil {
		panic(err)
	}

	logger.Info("Data loaded")

	var wg sync.WaitGroup
	globalReads := int64(0)
	globalHits := int64(0)
	globalCycles := int64(0)

	N := int64(1<<63 - 1)

	endtimes := time.NewTimer(runtime)
	go func() {
		<-endtimes.C
		logger.Info("Signalling the end times")
		N = -1
	}()

	logger.Info("Starting threads.", zap.Int("count", threads))
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			logger.Info("Starting thread", zap.Int("thread", i))
			var rnd = rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
			reads := int64(0)
			hits := int64(0)
			cycles := int64(0)
			for ; cycles < N; cycles++ {
				if cycles%1000000 == 0 {
					logger.Info(
						"Progress",
						zap.Int("thread", i),
						zap.Int64("count", cycles),
						zap.Int64("N", N),
					)
				}
				ix := rnd.Intn(count)
				if rnd.Float64() <= readRate {
					reads++
					if _, ok := lru.Get(keys[ix]); ok {
						hits++
					}
				} else {
					lru.Set(keys[ix], values[ix])
				}
			}
			atomic.AddInt64(&globalReads, reads)
			atomic.AddInt64(&globalHits, hits)
			atomic.AddInt64(&globalCycles, int64(cycles))
			logger.Info("Stopping thread", zap.Int("thread", i))
		}(i)
	}
	logger.Info("Waiting for threads to finish")
	wg.Wait()
	logger.Info("All threads finished. Closing lru")
	lru.Close()
	stats := lru.Stats()
	logger.Info(
		"stats",
		zap.Uint32("keys_written", stats.KeysWritten),
		zap.Uint32("read_ok", stats.KeysReadOK),
		zap.Uint32("read_not_found", stats.KeysReadNotFound),
		zap.Uint32("read_expired", stats.KeysReadExpired),
		zap.Uint32("shuffles", stats.Shuffles),
		zap.Uint32("evictions", stats.Evictions),
		zap.Uint32("reaped", stats.KeysReaped),
		zap.Uint32("reaper_cycles", stats.ReaperCycles),
	)
	logger.Info(
		"Done",
		zap.Int64("cycles", globalCycles),
		zap.Duration("runtime", runtime),
		zap.Float64("rate_kHz", float64(globalCycles)/(runtime.Seconds()*1000)),
		zap.Float64("hit_rate", float64(globalHits)*100.0/float64(globalReads)),
	)
}
