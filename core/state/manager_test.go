package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/idena-network/idena-go/config"
	"github.com/idena-network/idena-go/log"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"
)

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
