package sharded

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zeebo/xxh3"
)

func TestStrings(t *testing.T) {
	h := (*hasher)(xxh3.New())
	_, err := h.WriteString("x")
	require.NoError(t, err)
	v1 := h.Sum64()
	h.Reset()
	_, err = h.WriteString("x")
	require.NoError(t, err)
	v2 := h.Sum64()
	_, err = h.WriteString("x")
	require.NoError(t, err)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestBytes(t *testing.T) {
	h := (*hasher)(xxh3.New())
	_, err := h.Write([]byte("x"))
	require.NoError(t, err)
	v1 := h.Sum64()
	h.Reset()
	_, err = h.Write([]byte("x"))
	require.NoError(t, err)
	v2 := h.Sum64()
	_, err = h.Write([]byte("x"))
	require.NoError(t, err)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestUint8(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteUint8(1)
	v1 := h.Sum64()
	h.Reset()
	h.WriteUint8(1)
	v2 := h.Sum64()
	h.WriteUint8(1)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestUint16(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteUint16(1)
	v1 := h.Sum64()
	h.Reset()
	h.WriteUint16(1)
	v2 := h.Sum64()
	h.WriteUint16(1)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestUint32(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteUint32(1)
	v1 := h.Sum64()
	h.Reset()
	h.WriteUint32(1)
	v2 := h.Sum64()
	h.WriteUint32(1)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestUint64(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteUint64(1)
	v1 := h.Sum64()
	h.Reset()
	h.WriteUint64(1)
	v2 := h.Sum64()
	h.WriteUint64(1)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestFloat32(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteFloat32(1)
	v1 := h.Sum64()
	h.Reset()
	h.WriteFloat32(1)
	v2 := h.Sum64()
	h.WriteFloat32(1)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestFloat64(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteFloat64(1)
	v1 := h.Sum64()
	h.Reset()
	h.WriteFloat64(1)
	v2 := h.Sum64()
	h.WriteFloat64(1)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}

func TestBool(t *testing.T) {
	h := (*hasher)(xxh3.New())
	h.WriteBool(true)
	v1 := h.Sum64()
	h.Reset()
	h.WriteBool(true)
	v2 := h.Sum64()
	h.WriteBool(true)
	v3 := h.Sum64()
	require.Equal(t, v1, v2)
	require.NotEqual(t, v1, v3)
}
