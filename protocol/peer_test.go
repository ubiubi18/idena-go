package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("idena"), minCompressionSize)
	encoded := Encode(NewTx, payload)
	require.Equal(t, s2Compression, encoded[0])

	decoded, err := Decode(encoded)
	require.NoError(t, err)
	require.Equal(t, payload, decoded)
}

func TestDecodeRejectsOversizedCompressedMessageBeforeAllocation(t *testing.T) {
	encoded := []byte{s2Compression}
	encoded = binary.AppendUvarint(encoded, uint64(maxDecodedMsgSize)+1)

	decoded, err := Decode(encoded)
	require.ErrorContains(t, err, "exceeds limit")
	require.Nil(t, decoded)
}
