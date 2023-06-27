package main_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/TriggerMail/lazylru"
)

const keycnt = 100000

var keys = func() []string {
	k := make([]string, keycnt)

	for i := 0; i < keycnt; i++ {
		k[i] = strconv.Itoa(i)
	}
	return k
}()

type benchconfig struct {
	capacity int
	keyCount int
	readRate float64
}

func (bc benchconfig) Name() string {
	comment := "eqcap"
	if bc.capacity > bc.keyCount {
		comment = "overcap"
	} else if bc.capacity < bc.keyCount {
		comment = "undercap"
	}
	return fmt.Sprintf("%dW/%dR_%s", 100-int(100*bc.readRate), int(100*bc.readRate), comment)
}

func (bc benchconfig) Generic(b *testing.B) {
	lru := lazylru.NewT[string, int](bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], i)
	}

	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ix := rand.Intn(bc.keyCount)      //nolint:gosec
		if rand.Float64() < bc.readRate { //nolint:gosec
			lru.Get(keys[ix])
		} else {
			lru.Set(keys[ix], ix)
		}
	}
}

func (bc benchconfig) GenInterface(b *testing.B) {
	lru := lazylru.New(bc.capacity, time.Minute) //nolint:staticcheck
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], i)
	}

	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ix := rand.Intn(bc.keyCount)      //nolint:gosec
		if rand.Float64() < bc.readRate { //nolint:gosec
			lru.Get(keys[ix])
		} else {
			lru.Set(keys[ix], ix)
		}
	}
}

func Benchmark(b *testing.B) {
	for _, bc := range []benchconfig{
		{1, 1, 0.5}, // this is meant as a warm-up
		{1000, 100, 0.0},
		{1000, 100, 1.0},
		{1000, 100, 0.25},
		{1000, 100, 0.75},
		{1000, 100, 0.99},
		{100, 1000, 0.0},
		{100, 1000, 1.0},
		{100, 1000, 0.25},
		{100, 1000, 0.75},
		{100, 1000, 0.99},
		{100, 100, 0.0},
		{100, 100, 1.0},
		{100, 100, 0.25},
		{100, 100, 0.75},
		{100, 100, 0.99},
	} {
		b.Run(bc.Name()+"/gen[string,iface]", bc.GenInterface)
		b.Run(bc.Name()+"/gen[string,int]", bc.Generic)
	}
}
