package common

import (
	"bytes"
	"github.com/RoaringBitmap/roaring"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBitmap_Size(t *testing.T) {
	size := uint32(8888)

	rmap := roaring.NewBitmap()
	bitmap := NewBitmap(size)

	for i := uint32(0); i < size; i++ {
		if i%3 == 0 {
			rmap.Add(i)
			bitmap.Add(i)
		}
	}

	buf := new(bytes.Buffer)
	rmap.WriteTo(buf)

	buf2 := new(bytes.Buffer)
	bitmap.WriteTo(buf2)

	require.True(t, buf.Len() >= buf2.Len())
	require.Equal(t, buf2.Len(), int(size/8+1))

	rmap = roaring.NewBitmap()
	bitmap = NewBitmap(size)

	rmap.Add(0)
	bitmap.Add(0)
	rmap.Add(size - 1)
	bitmap.Add(size - 1)

	buf = new(bytes.Buffer)
	rmap.WriteTo(buf)

	buf2 = new(bytes.Buffer)
	bitmap.WriteTo(buf2)
	require.True(t, buf.Len()+1 >= buf2.Len())
}

func TestBitmap_Contains(t *testing.T) {
	size := uint32(8888)
	bitmap := NewBitmap(size)

	for i := uint32(0); i < size; i++ {
		if i%3 == 0 {
			bitmap.Add(i)
		}
	}

	buf := new(bytes.Buffer)
	bitmap.WriteTo(buf)

	require.NoError(t, bitmap.Read(buf.Bytes()))

	for i := uint32(0); i < size; i++ {
		if i%3 == 0 {
			require.True(t, bitmap.Contains(i))
		} else {
			require.False(t, bitmap.Contains(i))
		}
	}

	size = 2
	bitmap = NewBitmap(size)
	bitmap.Add(1)

	require.True(t, bitmap.Contains(1))

	buf = new(bytes.Buffer)
	bitmap.WriteTo(buf)

	require.NoError(t, bitmap.Read(buf.Bytes()))

	require.True(t, bitmap.Contains(1))

	require.False(t, bitmap.Contains(0))
}

func TestBitmap_Serialize(t *testing.T) {
	size := uint32(8)
	bitmap := NewBitmap(size)

	for i := uint32(0); i < size; i++ {
		if i%3 == 0 {
			bitmap.Add(i)
		}
	}

	buf := new(bytes.Buffer)
	bitmap.WriteTo(buf)

	bitmap2 := NewBitmap(size)
	require.NoError(t, bitmap2.Read(buf.Bytes()))

	require.True(t, bitmap.rmap.Equals(bitmap2.rmap))
}

func TestBitmap_SerializeSparseRoaringEncoding(t *testing.T) {
	const size = uint32(8888)
	bitmap := NewBitmap(size)
	bitmap.Add(0)
	bitmap.Add(size - 1)

	buf := new(bytes.Buffer)
	bitmap.WriteTo(buf)
	require.Equal(t, serializeDefault, buf.Bytes()[0])

	decoded := NewBitmap(size)
	require.NoError(t, decoded.Read(buf.Bytes()))
	require.True(t, bitmap.rmap.Equals(decoded.rmap))
}

func TestBitmap_ReadRejectsMalformedData(t *testing.T) {
	const size = uint32(8)

	outOfRange := roaring.NewBitmap()
	outOfRange.Add(size)
	outOfRangeData := new(bytes.Buffer)
	require.NoError(t, outOfRangeData.WriteByte(serializeDefault))
	_, err := outOfRange.WriteTo(outOfRangeData)
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty"},
		{name: "unknown encoding", data: []byte{0xff}},
		{name: "oversized encoding", data: []byte{serializeBigInt, 0, 0, 0}},
		{name: "truncated roaring bitmap", data: []byte{serializeDefault}},
		{name: "out of range roaring bitmap", data: outOfRangeData.Bytes()},
		{name: "out of range big integer", data: []byte{serializeBigInt, 0x01, 0x00}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bitmap := NewBitmap(size)
			bitmap.Add(1)

			require.Error(t, bitmap.Read(test.data))
			require.True(t, bitmap.Contains(1), "failed reads must not mutate the bitmap")
		})
	}
}
