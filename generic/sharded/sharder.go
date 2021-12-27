package sharded

import (
	"sync"

	"github.com/zeebo/xxh3"
)

// H is an interface to an underlying hasher
type H interface {
	Write(buf []byte) (int, error)
	WriteString(buf string) (int, error)
	WriteUint64(uint64)
	WriteUint32(uint32)
	WriteUint16(uint16)
	WriteUint8(uint8)
	WriteFloat32(float32)
	WriteFloat64(float64)
	WriteBool(bool)
}

// these pools allow the sharding operations to amortize to zero allocations
var (
	hpool = sync.Pool{New: func() any { return (*hasher)(xxh3.New()) }}
)

// StringSharder can be used to shard with string keys
var StringSharder = HashingSharder(func(k string, h H) {
	_, _ = h.WriteString(k)
})

// BytesSharder can be used to shard with byte slice keys
var BytesSharder = HashingSharder(func(k []byte, h H) {
	_, _ = h.Write(k)
})

// HashingSharder can be used to shard any type that can be written
// to a hasher
func HashingSharder[K any](f func(K, H)) func(K) uint64 {
	return func(key K) uint64 {
		h := hpool.Get().(*hasher)
		h.Reset()
		f(key, h)
		retval := h.Sum64()
		hpool.Put(h)
		return retval
	}
}
