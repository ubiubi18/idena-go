package subscriptions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/idena-network/idena-go/common"
)

func TestNewManagerTreatsEmptySubscriptionFileAsEmptyList(t *testing.T) {
	dir := t.TempDir()
	subscriptionsDir := filepath.Join(dir, Folder)
	if err := os.MkdirAll(subscriptionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subscriptionsDir, "subscriptions.json"), nil, 0o666); err != nil {
		t.Fatal(err)
	}

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got := len(manager.Subscriptions()); got != 0 {
		t.Fatalf("expected no subscriptions, got %d", got)
	}
}

func TestManagerPersistsSubscriptionsAfterEmptyFileLoad(t *testing.T) {
	dir := t.TempDir()
	subscriptionsDir := filepath.Join(dir, Folder)
	if err := os.MkdirAll(subscriptionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subscriptionsPath := filepath.Join(subscriptionsDir, "subscriptions.json")
	if err := os.WriteFile(subscriptionsPath, nil, 0o666); err != nil {
		t.Fatal(err)
	}

	manager, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.Subscribe(common.Address{0x01}, "Transfer"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(subscriptionsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `[{"contract":"0x0100000000000000000000000000000000000000","event":"Transfer"}]` {
		t.Fatalf("unexpected persisted subscriptions: %s", data)
	}
}
