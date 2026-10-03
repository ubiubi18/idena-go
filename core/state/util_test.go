package state

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/idena-network/idena-go/common"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"
)

func TestReadTreeFrom2RejectsOversizedSnapshotChunk(t *testing.T) {
	var input bytes.Buffer
	tw := tar.NewWriter(&input)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "0",
		Mode: 0600,
		Size: MaxSnapshotChunkBytes + 1,
	}))

	db := dbm.NewPrefixDB(dbm.NewMemDB(), []byte("snapshot"))
	err := ReadTreeFrom2(db, 1, common.Hash{}, &input)

	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds limit")
}

// The snapshot of a height must be the same bytes on every node: its CID comes from them, and nodes can
// only serve the pieces of a CID they have. The golden hash is WriteTreeTo2 of idena-network/idena-go
// v1.1.2 (mholt/archiver) for the same tree.
func TestWriteTreeTo2WritesTheSnapshotOfOfficialNodes(t *testing.T) {
	db := dbm.NewMemDB()
	tree := NewMutableTree(db)
	// More than SnapshotBlockSize nodes: two tar entries.
	for i := 0; i < SnapshotBlockSize+2345; i++ {
		tree.Set([]byte(fmt.Sprintf("key-%06d", i)), []byte(fmt.Sprintf("value-%d", i*7919)))
	}
	const height = 11369095
	_, _, err := tree.SaveVersionAt(height)
	require.NoError(t, err)

	var out bytes.Buffer
	_, err = WriteTreeTo2(db, height, &out)
	require.NoError(t, err)

	require.Equal(t, "89f140fb7620ea01aaf4af743c8bb6eb3304aa752c281a2aa73ee0320d67e105", fmt.Sprintf("%x", sha256.Sum256(out.Bytes())))
}
