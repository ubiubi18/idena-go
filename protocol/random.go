package protocol

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"math/big"
)

func secureIntn(n int) int {
	if n <= 0 {
		return 0
	}
	value, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(value.Int64())
}

func secureFloat32() float32 {
	var data [4]byte
	if _, err := crand.Read(data[:]); err != nil {
		return 0
	}
	return float32(float64(binary.LittleEndian.Uint32(data[:])) / (float64(math.MaxUint32) + 1))
}
