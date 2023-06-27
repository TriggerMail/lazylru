package sharded

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/zeebo/xxh3"
)

// these pools allow the sharding operations to amortize to zero allocations
var bufpool = sync.Pool{New: func() any { buf := make([]byte, 8); return &buf }}

type hasher xxh3.Hasher

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) Reset() {
	(*xxh3.Hasher)(h).Reset()
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) Sum64() uint64 {
	return (*xxh3.Hasher)(h).Sum64()
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) Write(buf []byte) (int, error) {
	return (*xxh3.Hasher)(h).Write(buf)
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteString(s string) (int, error) {
	return (*xxh3.Hasher)(h).WriteString(s)
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteUint64(v uint64) {
	bufP := bufpool.Get().(*[]byte)
	buf := *bufP
	binary.LittleEndian.PutUint64(buf, v)
	_, _ = h.Write(buf)
	bufpool.Put(bufP)
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteUint32(v uint32) {
	bufP := bufpool.Get().(*[]byte)
	buf := (*bufP)[0:4]
	binary.LittleEndian.PutUint32(buf, v)
	_, _ = h.Write(buf)
	bufpool.Put(bufP)
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteUint16(v uint16) {
	bufP := bufpool.Get().(*[]byte)
	buf := (*bufP)[0:2]
	binary.LittleEndian.PutUint16(buf, v)
	_, _ = h.Write(buf)
	bufpool.Put(bufP)
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteUint8(v uint8) {
	bufP := bufpool.Get().(*[]byte)
	buf := (*bufP)[0:1]
	buf[0] = v
	_, _ = h.Write(buf)
	bufpool.Put(bufP)
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteFloat32(v float32) {
	h.WriteUint32(math.Float32bits(v))
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteFloat64(v float64) {
	h.WriteUint64(math.Float64bits(v))
}

// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
func (h *hasher) WriteBool(v bool) {
	b := uint8(0)
	if v {
		b = 1
	}
	h.WriteUint8(b)
}
