package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomicCreatesAndReplacesPrivateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0644))

	require.NoError(t, WriteFileAtomic(path, []byte("new"), 0600))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, []byte("new"), data)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".secret.tmp-*"))
	require.NoError(t, err)
	require.Empty(t, tmpFiles)
}

func TestWriteFileAtomicRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires elevated privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "secret")
	require.NoError(t, os.WriteFile(target, []byte("unchanged"), 0600))
	require.NoError(t, os.Symlink(target, link))

	err := WriteFileAtomic(link, []byte("replacement"), 0600)

	require.ErrorContains(t, err, "non-regular file")
	data, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	require.Equal(t, []byte("unchanged"), data)
}

func TestWriteFileAtomicRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "secret")
	require.NoError(t, os.Mkdir(destination, 0700))

	err := WriteFileAtomic(destination, []byte("replacement"), 0600)

	require.ErrorContains(t, err, "non-regular file")
}
