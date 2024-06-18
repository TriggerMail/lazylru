package main

import (
	"math/rand/v2"
)

const validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(size int) string {
	retval := make([]byte, size)
	for i := 0; i < len(retval); i++ {
		retval[i] = validChars[rand.IntN(len(validChars))] //nolint:gosec
	}
	return string(retval)
}

// TestData represents a source of keys and values
type TestData interface {
	RandomKV() (string, string)
}

// KV is a key/value pair for use in the test
type KV struct {
	Key   string
	Value string
}

// TestDataSimple is a slice of KV
type TestDataSimple []KV

// NewTestData creates `count` KV instances
func NewTestData(count int) TestDataSimple {
	retval := make(TestDataSimple, count)

	for i := 0; i < count; i++ {
		retval[i] = KV{randomString(12), randomString(24)}
	}
	return retval
}

// RandomKV retrieves a random key/value pair
func (td TestDataSimple) RandomKV() (string, string) {
	kv := td[rand.IntN(len(td))] //nolint:gosec
	return kv.Key, kv.Value
}

// TestDataRanges is a series of test data sets that will be cycled through
// during a benchmark run
type TestDataRanges struct {
	ranges         []TestDataSimple
	cyclesPerRange int
	currentCycle   int
}

// TestDataSpec defined the shape of a benchmark run
type TestDataSpec struct {
	Ranges         int
	KeysPerRange   int
	CyclesPerRange int
}

// ToRanges creates TestDataRanges from a spec
func (tds TestDataSpec) ToRanges() *TestDataRanges {
	return NewTestDataRanges(tds.Ranges, tds.KeysPerRange, tds.CyclesPerRange)
}

// NewTestDataRanges creates test data ranges
func NewTestDataRanges(ranges int, keysPerRange int, cyclesPerRange int) *TestDataRanges {
	td := make([]TestDataSimple, ranges)
	for i := 0; i < ranges; i++ {
		td[i] = NewTestData(keysPerRange)
	}
	return &TestDataRanges{
		cyclesPerRange: cyclesPerRange,
		ranges:         td,
	}
}

// RandomKV pulls a random item from the current range in a benchmark run
func (tdr *TestDataRanges) RandomKV() (string, string) {
	tdr.currentCycle++
	return tdr.ranges[tdr.currentCycle%len(tdr.ranges)].RandomKV()
}
