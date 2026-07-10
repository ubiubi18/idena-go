package common

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/RoaringBitmap/roaring"
)

const (
	serializeDefault = byte(0x1)
	serializeBigInt  = byte(0x2)
)

type Bitmap struct {
	rmap *roaring.Bitmap
	size uint32
}

func NewBitmap(size uint32) *Bitmap {
	return &Bitmap{size: size, rmap: roaring.NewBitmap()}
}

func (m *Bitmap) Add(value uint32) {
	if value >= m.size {
		panic("value is out of range")
	}
	m.rmap.Add(value)

}

func (m *Bitmap) Contains(value uint32) bool {
	return m.rmap.Contains(value)
}

func (m *Bitmap) WriteTo(buffer *bytes.Buffer) {
	if m.rmap.HasRunCompression() {
		m.rmap.RunOptimize()
	}
	if m.rmap.GetSerializedSizeInBytes() > uint64(m.size/8+1) {
		bits := big.NewInt(0)
		buffer.WriteByte(serializeBigInt)
		for _, v := range m.rmap.ToArray() {
			t := big.NewInt(1)
			bits.Or(bits, t.Lsh(t, uint(v)))
		}
		buffer.Write(bits.Bytes())
	} else {
		buffer.WriteByte(serializeDefault)
		m.rmap.WriteTo(buffer)
	}
}

func (m *Bitmap) Read(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("bitmap encoding is empty")
	}
	if len(data) > int(m.size/8)+2 {
		return fmt.Errorf("bitmap encoding is too large for size %d", m.size)
	}

	rmap := roaring.NewBitmap()
	switch data[0] {
	case serializeDefault:
		buf := bytes.NewReader(data[1:])
		if _, err := rmap.ReadFrom(buf); err != nil {
			return fmt.Errorf("decode roaring bitmap: %w", err)
		}
		if buf.Len() != 0 {
			return fmt.Errorf("bitmap encoding has %d trailing bytes", buf.Len())
		}
		if !rmap.IsEmpty() && rmap.Maximum() >= m.size {
			return fmt.Errorf("bitmap value %d exceeds size %d", rmap.Maximum(), m.size)
		}
	case serializeBigInt:
		bits := new(big.Int).SetBytes(data[1:])
		if bits.BitLen() > int(m.size) {
			return fmt.Errorf("bitmap value exceeds size %d", m.size)
		}
		for i := uint32(0); i < m.size; i++ {
			if bits.Bit(int(i)) == 1 {
				rmap.Add(i)
			}
		}
	default:
		return fmt.Errorf("unknown bitmap encoding 0x%x", data[0])
	}

	m.rmap = rmap
	return nil
}

func (m *Bitmap) ToArray() []uint32 {
	return m.rmap.ToArray()
}
