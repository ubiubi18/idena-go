package subscriptions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/idena-network/idena-go/common"
	"github.com/stretchr/testify/require"
)

func TestNewManagerIgnoresOpenFileError(t *testing.T) {
	datadir := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(datadir, []byte("file"), 0600))

	manager, err := NewManager(datadir)

	require.NoError(t, err)
	require.NotNil(t, manager)
	require.Error(t, manager.Subscribe(common.Address{0x1}, "event"))
}

func TestManagerOpenFileCreatesPrivateStorage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX file mode bits")
	}

	datadir := t.TempDir()
	manager := &Manager{datadir: datadir}

	file, err := manager.openFile()
	require.NoError(t, err)
	require.NoError(t, file.Close())

	assertMode(t, filepath.Join(datadir, Folder), 0700)
	assertMode(t, filepath.Join(datadir, Folder, "subscriptions.json"), 0600)
}

func assertMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, mode, info.Mode().Perm())
}
