package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/bits"
	"sort"

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

// LegacyBitmapVotes decodes evidence maps with the original consensus rules.
// Unlike Read, every non-Roaring prefix is a big integer. Roaring containers
// are counted directly so out-of-range runs cannot expand into huge arrays.
func LegacyBitmapVotes(data []byte, size uint32) (map[uint32]int, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("bitmap encoding is empty")
	}

	if data[0] != serializeDefault {
		votes := make(map[uint32]int)
		payload := data[1:]
		for i := uint32(0); i < size; i++ {
			byteIndex := int(i / 8)
			if byteIndex >= len(payload) {
				break
			}
			if payload[len(payload)-1-byteIndex]&(1<<(i%8)) != 0 {
				votes[i]++
			}
		}
		return votes, nil
	}
	return legacyRoaringVotes(data[1:], size)
}

// legacyRoaringVotes follows the Roaring v0.9.4 wire format consumed by the
// official node. ToArray emits each container's values (including duplicates)
// and pads the remainder of its declared cardinality with zeroes. Count those
// votes without materializing the array or iterating over out-of-range runs.
func legacyRoaringVotes(data []byte, size uint32) (map[uint32]int, error) {
	position := 0
	read := func(length int) ([]byte, error) {
		if length > len(data)-position {
			return nil, fmt.Errorf("truncated roaring bitmap")
		}
		part := data[position : position+length]
		position += length
		return part, nil
	}

	header, err := read(4)
	if err != nil {
		return nil, err
	}
	cookie := binary.LittleEndian.Uint32(header)
	var containerCount uint32
	var runFlags []byte
	switch {
	case cookie&0xffff == 12347:
		containerCount = cookie>>16 + 1
		runFlags, err = read(int((containerCount + 7) / 8))
	case cookie == 12346:
		var count []byte
		count, err = read(4)
		if err == nil {
			containerCount = binary.LittleEndian.Uint32(count)
		}
	default:
		return nil, fmt.Errorf("unknown roaring bitmap cookie")
	}
	if err != nil {
		return nil, err
	}
	if containerCount > 1<<16 {
		return nil, fmt.Errorf("too many roaring containers")
	}
	descriptors, err := read(int(containerCount) * 4)
	if err != nil {
		return nil, err
	}
	if runFlags == nil || containerCount >= 4 {
		if _, err = read(int(containerCount) * 4); err != nil {
			return nil, err
		}
	}

	votes := make(map[uint32]int)
	rangeChanges := make(map[uint32]int64)
	addRange := func(first, last uint32) {
		if first >= size {
			return
		}
		if last >= size {
			last = size - 1
		}
		rangeChanges[first]++
		if last+1 < size {
			rangeChanges[last+1]--
		}
	}
	var declared, actual int64
	for i := 0; i < int(containerCount); i++ {
		key := uint32(binary.LittleEndian.Uint16(descriptors[i*4:]))
		cardinality := int(binary.LittleEndian.Uint16(descriptors[i*4+2:])) + 1
		base := key << 16
		isRun := runFlags != nil && runFlags[i/8]&(1<<(uint(i)%8)) != 0
		switch {
		case isRun:
			countBytes, readErr := read(2)
			if readErr != nil {
				return nil, readErr
			}
			runCount := int(binary.LittleEndian.Uint16(countBytes))
			body, readErr := read(runCount * 4)
			if readErr != nil {
				return nil, readErr
			}
			for j := 0; j < runCount; j++ {
				start := uint32(binary.LittleEndian.Uint16(body[j*4:]))
				length := uint32(binary.LittleEndian.Uint16(body[j*4+2:]))
				end := start + length
				declared += int64(length) + 1
				actual += int64(length) + 1
				if end < 1<<16 {
					addRange(base|start, base|end)
				} else {
					addRange(base|start, base|0xffff)
					addRange(((key | 1) << 16), ((key|1)<<16)|(end-(1<<16)))
				}
			}
		case cardinality > 4096:
			body, readErr := read(8192)
			if readErr != nil {
				return nil, readErr
			}
			declared += int64(cardinality)
			for j, value := range body {
				actual += int64(bits.OnesCount8(value))
				if base >= size || value == 0 {
					continue
				}
				for bit := uint32(0); bit < 8; bit++ {
					index := base | uint32(j*8) | bit
					if index < size && value&(1<<bit) != 0 {
						votes[index]++
					}
				}
			}
		default:
			body, readErr := read(cardinality * 2)
			if readErr != nil {
				return nil, readErr
			}
			declared += int64(cardinality)
			actual += int64(cardinality)
			for j := 0; j < cardinality; j++ {
				if index := base | uint32(binary.LittleEndian.Uint16(body[j*2:])); index < size {
					votes[index]++
				}
			}
		}
	}
	if actual > declared || declared > int64(int(^uint(0)>>1)) {
		return nil, fmt.Errorf("roaring bitmap cardinality cannot be materialized by legacy node")
	}
	if size > 0 {
		if declared > actual {
			votes[0] += int(declared - actual)
		}
		if len(rangeChanges) > 0 {
			boundaries := make([]uint32, 0, len(rangeChanges))
			for index := range rangeChanges {
				boundaries = append(boundaries, index)
			}
			sort.Slice(boundaries, func(i, j int) bool { return boundaries[i] < boundaries[j] })
			var current int64
			for i, first := range boundaries {
				current += rangeChanges[first]
				last := size
				if i+1 < len(boundaries) {
					last = boundaries[i+1]
				}
				if current != 0 {
					for index := first; index < last; index++ {
						votes[index] += int(current)
					}
				}
			}
		}
	}
	return votes, nil
}

func (m *Bitmap) ToArray() []uint32 {
	return m.rmap.ToArray()
}
