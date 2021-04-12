package main

import (
	"math/rand"
)

const validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(size int) string {
	retval := make([]byte, size)
	for i := 0; i < len(retval); i++ {
		retval[i] = validChars[rand.Intn(len(validChars))]
	}
	return string(retval)
}

type TestData interface {
	RandomKV() (string, interface{})
}

type KV struct {
	Key   string
	Value interface{}
}

type TestDataSimple []KV

func NewTestData(count int) TestDataSimple {
	retval := make(TestDataSimple, count)

	for i := 0; i < count; i++ {
		retval[i] = KV{randomString(12), randomString(24)}
	}
	return retval
}

func (td TestDataSimple) RandomKV() (string, interface{}) {
	kv := td[rand.Intn(len(td))]
	return kv.Key, kv.Value
}

type TestDataRanges struct {
	cyclesPerRange int
	currentCycle   int
	ranges         []TestDataSimple
}

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

func (tdr *TestDataRanges) RandomKV() (string, interface{}) {
	tdr.currentCycle++
	return tdr.ranges[tdr.currentCycle%len(tdr.ranges)].RandomKV()
}
