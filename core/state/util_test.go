package state

import (
	"archive/tar"
	"bytes"
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
