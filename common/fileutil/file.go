package fileutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsurePrivateDir creates path if needed, rejects links and non-directories,
// and restricts the opened directory to its owner.
func EnsurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("private directory path is not a directory: %q", path)
	}

	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	openedInfo, err := dir.Stat()
	if err != nil {
		return err
	}
	if !openedInfo.IsDir() || !os.SameFile(info, openedInfo) {
		return fmt.Errorf("private directory path changed while opening: %q", path)
	}
	return dir.Chmod(0700)
}

// WriteFileAtomic replaces path with data only after the new contents have
// been written and flushed. Existing non-regular files are never followed or
// replaced.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if err := validateDestination(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(perm.Perm()); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return ReplaceFileAtomic(tmpPath, path)
}

// ReplaceFileAtomic moves a completed temporary file into place and flushes
// the containing directory. Both paths must name regular files in one directory.
func ReplaceFileAtomic(tempPath, path string) error {
	if filepath.Clean(filepath.Dir(tempPath)) != filepath.Clean(filepath.Dir(path)) {
		return fmt.Errorf("temporary file must be in the destination directory")
	}
	tempInfo, err := os.Lstat(tempPath)
	if err != nil {
		return err
	}
	if !tempInfo.Mode().IsRegular() {
		return fmt.Errorf("temporary path is not a regular file: %q", tempPath)
	}
	if err := validateDestination(path); err != nil {
		return err
	}
	if err := replaceFile(tempPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func validateDestination(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to replace non-regular file %q", path)
	}
	return nil
}
