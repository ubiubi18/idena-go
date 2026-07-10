//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package rpc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIPCListenerRestrictsSocketPermissions(t *testing.T) {
	dir, err := os.MkdirTemp("", "idena-ipc-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})
	endpoint := filepath.Join(dir, "node.ipc")
	listener, err := ipcListen(endpoint)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})

	info, err := os.Stat(endpoint)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
