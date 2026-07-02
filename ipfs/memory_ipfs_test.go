//go:build idena_memory_ipfs

package ipfs

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryIpfsProxyStoresCopies(t *testing.T) {
	proxy := NewMemoryIpfsProxy()
	data := []byte("contract payload")

	c, err := proxy.Add(data, true)
	require.NoError(t, err)

	data[0] = 'C'
	stored, err := proxy.Get(c.Bytes(), Block)
	require.NoError(t, err)
	require.Equal(t, []byte("contract payload"), stored)

	stored[0] = 'X'
	storedAgain, err := proxy.Get(c.Bytes(), Block)
	require.NoError(t, err)
	require.Equal(t, []byte("contract payload"), storedAgain)
}

func TestMemoryIpfsProxyEmptyDataUsesEmptyCid(t *testing.T) {
	proxy := NewMemoryIpfsProxy()

	c, err := proxy.Add(nil, true)
	require.NoError(t, err)
	require.Equal(t, EmptyCid, c)

	data, err := proxy.Get(c.Bytes(), Block)
	require.NoError(t, err)
	require.Empty(t, data)
}

func TestMemoryIpfsProxyHonorsExplicitSizeLimit(t *testing.T) {
	proxy := NewMemoryIpfsProxy()
	c, err := proxy.Add([]byte("12345"), true)
	require.NoError(t, err)

	_, err = proxy.GetWithSizeLimit(c.Bytes(), Block, 4)
	require.ErrorIs(t, err, TooBigErr)

	data, err := proxy.GetWithSizeLimit(c.Bytes(), Block, 5)
	require.NoError(t, err)
	require.Equal(t, []byte("12345"), data)
}

func TestMemoryIpfsProxyLoadTo(t *testing.T) {
	proxy := NewMemoryIpfsProxy()
	c, err := proxy.Add([]byte("snapshot"), true)
	require.NoError(t, err)

	var buf bytes.Buffer
	var progress []int64
	err = proxy.LoadTo(c.Bytes(), &buf, context.Background(), func(size, loaded int64) {
		require.Equal(t, int64(len("snapshot")), size)
		progress = append(progress, loaded)
	})
	require.NoError(t, err)
	require.Equal(t, "snapshot", buf.String())
	require.Equal(t, []int64{0, int64(len("snapshot"))}, progress)
}
