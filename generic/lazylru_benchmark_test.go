package lazylru_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"

	lazylru "github.com/TriggerMail/lazylru/generic"
)

const keycnt = 100000

type value struct {
	x *int
	s string
	v int
	b byte
}

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

func (bc benchconfig) InterfaceArray(b *testing.B) {
	lru := lazylru.New(bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], []int{i})
	}
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ix := rand.Intn(bc.keyCount)      //nolint:gosec
		ix := i % bc.keyCount
		if rand.Float64() < bc.readRate { //nolint:gosec
			// if true {
			if iv, ok := lru.Get(keys[ix]); !ok {
				continue
			} else {
				v, ok := iv.([]int)
				if !ok {
					b.Fatalf("expected integer value, got %v", iv)
				}
				if v[0] != ix {
					b.Fatalf("expected %d, got %d", ix, v[0])
				}
			}
		} else {
			lru.Set(keys[ix], []int{ix})
		}
	}
}

func (bc benchconfig) GenericArray(b *testing.B) {
	lru := lazylru.NewT[string, []int](bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], []int{i})
	}
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ix := rand.Intn(bc.keyCount)      //nolint:gosec
		ix := i % bc.keyCount
		if rand.Float64() < bc.readRate { //nolint:gosec
			// if true {
			if v, ok := lru.Get(keys[ix]); !ok {
				continue
			} else if v[0] != ix {
				b.Fatalf("expected %d, got %d", ix, v[0])
			}
		} else {
			lru.Set(keys[ix], []int{ix})
		}
	}
}

func (bc benchconfig) InterfaceStructPtr(b *testing.B) {
	lru := lazylru.New(bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], &value{v: i})
	}
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ix := rand.Intn(bc.keyCount)      //nolint:gosec
		ix := i % bc.keyCount
		if rand.Float64() < bc.readRate { //nolint:gosec
			// if true {
			if iv, ok := lru.Get(keys[ix]); !ok {
				continue
			} else {
				v, ok := iv.(*value)
				if !ok {
					b.Fatalf("expected integer value, got %v", iv)
				}
				if v.v != ix {
					b.Fatalf("expected %d, got %d", ix, v.v)
				}
			}
		} else {
			lru.Set(keys[ix], &value{v: ix})
		}
	}
}

func (bc benchconfig) GenericStructPtr(b *testing.B) {
	lru := lazylru.NewT[string, *value](bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], &value{v: i})
	}
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ix := rand.Intn(bc.keyCount)      //nolint:gosec
		ix := i % bc.keyCount
		if rand.Float64() < bc.readRate { //nolint:gosec
			// if true {
			if v, ok := lru.Get(keys[ix]); !ok {
				continue
			} else if v.v != ix {
				b.Fatalf("expected %d, got %d", ix, v.v)
			}
		} else {
			lru.Set(keys[ix], &value{v: ix})
		}
	}
}

func (bc benchconfig) InterfaceValue(b *testing.B) {
	lru := lazylru.New(bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], i)
	}
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ix := rand.Intn(bc.keyCount)      //nolint:gosec
		ix := i % bc.keyCount
		if rand.Float64() < bc.readRate { //nolint:gosec
			// if true {
			if iv, ok := lru.Get(keys[ix]); !ok {
				continue
			} else {
				v, ok := iv.(int)
				if !ok {
					b.Fatalf("expected integer value, got %v", iv)
				}
				if v != ix {
					b.Fatalf("expected %d, got %d", ix, v)
				}
			}
		} else {
			lru.Set(keys[ix], ix)
		}
	}
}

func (bc benchconfig) GenericValue(b *testing.B) {
	lru := lazylru.NewT[string, int](bc.capacity, time.Minute)
	defer lru.Close()
	for i := 0; i < bc.keyCount; i++ {
		lru.Set(keys[i], i)
	}
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// ix := rand.Intn(bc.keyCount)      //nolint:gosec
		ix := i % bc.keyCount
		if rand.Float64() < bc.readRate { //nolint:gosec
			// if true {
			if v, ok := lru.Get(keys[ix]); !ok {
				continue
			} else if v != ix {
				b.Fatalf("expected %d, got %d", ix, v)
			}
		} else {
			lru.Set(keys[ix], ix)
		}
	}
}

func Benchmark(b *testing.B) {
	for _, bc := range []benchconfig{
		// {1, 1, 0.5}, // this is meant as a warm-up
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
		b.Run(bc.Name()+"/interface/array", bc.InterfaceArray)
		b.Run(bc.Name()+"/generic/array", bc.GenericArray)
		b.Run(bc.Name()+"/interface/struct", bc.InterfaceStructPtr)
		b.Run(bc.Name()+"/generic/struct", bc.GenericStructPtr)
		b.Run(bc.Name()+"/interface/value", bc.InterfaceValue)
		b.Run(bc.Name()+"/generic/value", bc.GenericValue)
	}
}
