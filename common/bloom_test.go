package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSerializableBF_Has(t *testing.T) {
	b := NewSerializableBF(100)
	addr1 := Address{0x1}
	addr2 := Address{0x2}
	addr3 := Address{0x3}
	addr4 := Address{0x4}

	b.Add(addr1.Bytes())
	b.Add(addr2.Bytes())
	b.Add(addr3.Bytes())
	b.Add(addr4.Bytes())

	bloomData, err := b.Serialize()
	require.NoError(t, err)

	b2, err := NewSerializableBFFromData(bloomData)

	require.NoError(t, err)
	require.Equal(t, b.data, b2.data)

	require.True(t, b2.Has(addr1.Bytes()))
	require.True(t, b2.Has(addr2.Bytes()))
	require.True(t, b2.Has(addr3.Bytes()))
	require.True(t, b2.Has(addr4.Bytes()))
	require.False(t, b2.Has(Address{0x5}.Bytes()))
}

func TestSerializableBFRejectsMalformedData(t *testing.T) {
	for _, length := range []int{0, 1, 7, 9, 24, 513} {
		filter, err := NewSerializableBFFromData(make([]byte, length))
		require.Error(t, err, "length %d", length)
		require.Nil(t, filter, "length %d", length)
	}
}

func TestSerializableBFAcceptsProducedSizes(t *testing.T) {
	for _, count := range []int{0, 8, 16, 32, 64, 128, 256} {
		data, err := NewSerializableBF(count).Serialize()
		require.NoError(t, err)
		filter, err := NewSerializableBFFromData(data)
		require.NoError(t, err, "count %d", count)
		require.NotNil(t, filter)
	}
}
