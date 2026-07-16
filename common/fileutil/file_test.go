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

func TestEnsurePrivateDirTightensExistingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX directory mode bits")
	}

	path := filepath.Join(t.TempDir(), "keystore")
	require.NoError(t, os.Mkdir(path, 0755))

	require.NoError(t, EnsurePrivateDir(path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestEnsurePrivateDirRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires elevated privileges on Windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "keystore")
	require.NoError(t, os.Mkdir(target, 0755))
	require.NoError(t, os.Symlink(target, link))

	err := EnsurePrivateDir(link)

	require.ErrorContains(t, err, "not a directory")
	info, statErr := os.Stat(target)
	require.NoError(t, statErr)
	require.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestReplaceFileAtomicCommitsSameDirectoryFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "key")
	temporary := filepath.Join(dir, ".key.tmp")
	require.NoError(t, os.WriteFile(destination, []byte("old"), 0600))
	require.NoError(t, os.WriteFile(temporary, []byte("new"), 0600))

	require.NoError(t, ReplaceFileAtomic(temporary, destination))

	data, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, []byte("new"), data)
	require.NoFileExists(t, temporary)
}

func TestReplaceFileAtomicRejectsDifferentDirectory(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	destinationDir := filepath.Join(root, "destination")
	require.NoError(t, os.Mkdir(sourceDir, 0700))
	require.NoError(t, os.Mkdir(destinationDir, 0700))
	temporary := filepath.Join(sourceDir, "key.tmp")
	destination := filepath.Join(destinationDir, "key")
	require.NoError(t, os.WriteFile(temporary, []byte("new"), 0600))

	err := ReplaceFileAtomic(temporary, destination)

	require.ErrorContains(t, err, "destination directory")
	require.FileExists(t, temporary)
	require.NoFileExists(t, destination)
}
