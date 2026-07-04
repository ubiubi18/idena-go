package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/idena-network/idena-go/config"
	"github.com/stretchr/testify/require"
)

func TestGetLogFileHandlerReturnsOpenError(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	require.NoError(t, os.WriteFile(dataDir, []byte("not a directory"), 0600))

	_, err := getLogFileHandler(&config.Config{DataDir: dataDir}, 1)

	require.Error(t, err)
}
