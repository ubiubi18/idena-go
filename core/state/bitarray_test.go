package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitArrayUnmarshalJSONUpdatesFields(t *testing.T) {
	bits := NewBitArray(1)

	require.NoError(t, json.Unmarshal([]byte(`"x_x_"`), bits))

	require.Equal(t, uint(4), bits.Size())
	require.True(t, bits.GetIndex(0))
	require.False(t, bits.GetIndex(1))
	require.True(t, bits.GetIndex(2))
	require.False(t, bits.GetIndex(3))
}

func TestBitArrayUnmarshalJSONNullResetsFields(t *testing.T) {
	bits := NewBitArray(4)
	bits.SetIndex(1, true)

	require.NoError(t, json.Unmarshal([]byte(`null`), bits))

	require.Zero(t, bits.Size())
	require.False(t, bits.GetIndex(1))
}
