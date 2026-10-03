package state

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/eventbus"
	"github.com/idena-network/idena-go/config"
	"github.com/idena-network/idena-go/core/state/snapshot"
	"github.com/idena-network/idena-go/events"
	"github.com/idena-network/idena-go/ipfs"
	"github.com/idena-network/idena-go/log"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"
)

type snapshotDownloadProxy struct{ ipfs.Proxy }

func (*snapshotDownloadProxy) LoadTo(_ []byte, to io.Writer, _ context.Context, onLoading func(int64, int64)) error {
	data := []byte("unverified snapshot")
	n, err := to.Write(data)
	onLoading(int64(len(data)), int64(n))
	return err
}

func (*snapshotDownloadProxy) Unpin([]byte) error { return nil }

func TestSnapshotManager_IsInvalidManifest(t *testing.T) {
	m := SnapshotManager{
		db: db.NewMemDB(),
	}
	m.AddInvalidManifest([]byte{0x1})

	m.AddTimeoutManifest([]byte{0x3})
	m.AddTimeoutManifest([]byte{0x3})
	m.AddTimeoutManifest([]byte{0x3})
	m.AddTimeoutManifest([]byte{0x3})
	m.AddTimeoutManifest([]byte{0x3})

	m.AddTimeoutManifest([]byte{0x4})
	m.AddTimeoutManifest([]byte{0x4})
	m.AddTimeoutManifest([]byte{0x4})
	m.AddTimeoutManifest([]byte{0x4})

	require.True(t, m.IsInvalidManifest([]byte{0x1}))
	require.False(t, m.IsInvalidManifest([]byte{0x2}))
	require.True(t, m.IsInvalidManifest([]byte{0x3}))
	require.False(t, m.IsInvalidManifest([]byte{0x4}))
}

func TestStoreSnapshotManifestAnnouncesStoredSnapshot(t *testing.T) {
	bus := eventbus.New()
	datadir := t.TempDir()
	m := NewSnapshotManager(db.NewMemDB(), nil, bus, nil, &config.Config{DataDir: datadir})
	filePath, file, err := createSnapshotFile(datadir, 1000, SnapshotVersionV2)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	want := &snapshot.Manifest{CidV2: []byte{1, 2, 3}, Root: common.Hash{4}, Height: 1000}
	var announced *snapshot.Manifest
	bus.Subscribe(events.NewSnapshotManifestEventID, func(e eventbus.Event) {
		announced = e.(*events.NewSnapshotManifestEvent).Manifest
		cid, root, height, _ := m.repo.LastSnapshotManifest()
		require.Equal(t, want.CidV2, cid)
		require.Equal(t, want.Root, root)
		require.Equal(t, want.Height, height)
	})

	m.StoreSnapshotManifest(want, filePath)
	require.Equal(t, want, announced)
}

func TestDownloadedSnapshotIsNotAdvertisedBeforeValidation(t *testing.T) {
	bus := eventbus.New()
	proxy := &snapshotDownloadProxy{}
	datadir := t.TempDir()
	m := NewSnapshotManager(db.NewMemDB(), nil, bus, proxy, &config.Config{DataDir: datadir})
	announcements := 0
	bus.Subscribe(events.NewSnapshotManifestEventID, func(eventbus.Event) {
		announcements++
	})
	previous := &snapshot.Manifest{CidV2: []byte{9}, Root: common.Hash{8}, Height: 900}
	previousPath, file, err := createSnapshotFile(datadir, previous.Height, SnapshotVersionV2)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	m.StoreSnapshotManifest(previous, previousPath)
	require.Equal(t, 1, announcements)
	announcements = 0

	manifest := &snapshot.Manifest{CidV2: []byte{1, 2, 3}, Root: common.Hash{4}, Height: 1000}
	filePath, _, err := m.DownloadSnapshot(manifest)
	require.NoError(t, err)
	_, err = os.Stat(filePath)
	require.NoError(t, err)

	storedCID, _, _, _ := m.repo.LastSnapshotManifest()
	require.Equal(t, previous.CidV2, storedCID)
	require.Zero(t, announcements)

	m.StoreSnapshotManifest(manifest, filePath)
	storedCID, _, _, _ = m.repo.LastSnapshotManifest()
	require.Equal(t, manifest.CidV2, storedCID)
	require.Equal(t, 1, announcements)
	_, err = os.Stat(previousPath)
	require.True(t, os.IsNotExist(err))
}

func TestCreateSnapshotFileCreatesPrivateStorage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX file mode bits")
	}

	datadir := t.TempDir()

	fileName, file, err := createSnapshotFile(datadir, 123, SnapshotVersionV2)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	require.True(t, strings.HasPrefix(fileName, filepath.Join(datadir, SnapshotsFolder)))
	assertMode(t, filepath.Join(datadir, SnapshotsFolder), 0700)
	assertMode(t, fileName, 0600)
}

func TestClearSnapshotFolderHandlesMissingDirectory(t *testing.T) {
	m := SnapshotManager{
		cfg: &config.Config{DataDir: t.TempDir()},
		log: log.New(),
	}

	require.NotPanics(t, func() {
		m.clearSnapshotFolder(nil)
	})
}

func assertMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, mode, info.Mode().Perm())
}
