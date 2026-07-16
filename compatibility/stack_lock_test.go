package compatibility_test

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/idena-network/idena-go/blockchain"
	"github.com/idena-network/idena-go/protocol"
)

const (
	wantReleaseID             = "idena-mainnet-legacy-compat-2026.07.12-rc3"
	wantNodeCommit            = "aafb254786ac3c82308550a7a82642019f077d6b"
	wantRuntimeCommit         = "aafb254786ac3c82308550a7a82642019f077d6b"
	wantBindingCommit         = "67ba065fdb02aa07cced2a43a261e481ca5b39d9"
	wantGossipProtocol        = "/idena/gossip/1.1.0"
	wantMainnetNetwork uint32 = 1
)

type component struct {
	Name              string `json:"name"`
	Commit            string `json:"commit"`
	RuntimeCodeCommit string `json:"runtimeCodeCommit"`
}

type gateResult struct {
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
	SHA256   string `json:"sha256"`
}

type stackLock struct {
	Schema          int    `json:"schema"`
	ReleaseID       string `json:"releaseId"`
	Status          string `json:"status"`
	ChainInvariants struct {
		MainnetNetworkID        uint32 `json:"mainnetNetworkId"`
		GossipProtocol          string `json:"gossipProtocol"`
		ConsensusChangesAllowed bool   `json:"consensusChangesAllowed"`
	} `json:"chainInvariants"`
	Components    []component           `json:"components"`
	RequiredGates []string              `json:"requiredGates"`
	GateResults   map[string]gateResult `json:"gateResults"`
}

func TestReleaseApprovalRequiresEvidenceForEveryGate(t *testing.T) {
	lock := loadLock(t)
	if lock.Status != "candidate" && lock.Status != "approved" {
		t.Fatalf("unsupported compatibility lock status %q", lock.Status)
	}
	if len(lock.RequiredGates) == 0 {
		t.Fatal("compatibility lock has no required gates")
	}
	seen := make(map[string]struct{}, len(lock.RequiredGates))
	sha256 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for _, gate := range lock.RequiredGates {
		if _, exists := seen[gate]; exists {
			t.Fatalf("duplicate required gate %q", gate)
		}
		seen[gate] = struct{}{}
		if lock.Status != "approved" {
			continue
		}
		result, exists := lock.GateResults[gate]
		if !exists || result.Status != "passed" {
			t.Fatalf("approved lock has no passing result for %q", gate)
		}
		if len(result.Evidence) < len("https://") || result.Evidence[:len("https://")] != "https://" {
			t.Fatalf("approved gate %q has no HTTPS evidence URL", gate)
		}
		if !sha256.MatchString(result.SHA256) {
			t.Fatalf("approved gate %q has no evidence digest", gate)
		}
	}
	for gate := range lock.GateResults {
		if _, exists := seen[gate]; !exists {
			t.Fatalf("result provided for unrequired gate %q", gate)
		}
	}
}

func loadLock(t *testing.T) stackLock {
	t.Helper()
	raw, err := os.ReadFile("stack-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	var lock stackLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	return lock
}

func TestStackLockPinsReviewedRuntime(t *testing.T) {
	lock := loadLock(t)
	if lock.Schema != 1 || lock.ReleaseID != wantReleaseID || lock.Status != "candidate" {
		t.Fatalf("unexpected compatibility lock identity: schema=%d release=%q status=%q", lock.Schema, lock.ReleaseID, lock.Status)
	}
	components := make(map[string]component, len(lock.Components))
	for _, item := range lock.Components {
		if _, exists := components[item.Name]; exists {
			t.Fatalf("duplicate component %q", item.Name)
		}
		components[item.Name] = item
	}
	if got := components["idena-go"]; got.Commit != wantNodeCommit || got.RuntimeCodeCommit != wantRuntimeCommit {
		t.Fatalf("idena-go lock drifted: commit=%q runtime=%q", got.Commit, got.RuntimeCodeCommit)
	}
	if got := components["idena-wasm-binding"].Commit; got != wantBindingCommit {
		t.Fatalf("binding lock drifted: %q", got)
	}
	sha1 := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for name, item := range components {
		if !sha1.MatchString(item.Commit) {
			t.Fatalf("component %q has an invalid commit", name)
		}
	}
}

func TestCompiledChainIdentifiersMatchLock(t *testing.T) {
	lock := loadLock(t)
	if lock.ChainInvariants.ConsensusChangesAllowed {
		t.Fatal("compatibility candidate permits consensus changes")
	}
	if lock.ChainInvariants.MainnetNetworkID != wantMainnetNetwork || uint32(blockchain.Mainnet) != wantMainnetNetwork {
		t.Fatalf("mainnet network ID drifted: lock=%d binary=%d", lock.ChainInvariants.MainnetNetworkID, blockchain.Mainnet)
	}
	if lock.ChainInvariants.GossipProtocol != wantGossipProtocol || string(protocol.IdenaProtocol) != wantGossipProtocol {
		t.Fatalf("gossip protocol drifted: lock=%q binary=%q", lock.ChainInvariants.GossipProtocol, protocol.IdenaProtocol)
	}
}
