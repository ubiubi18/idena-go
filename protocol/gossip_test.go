package protocol

import (
	"testing"

	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/eventbus"
	"github.com/idena-network/idena-go/core/state/snapshot"
	"github.com/idena-network/idena-go/events"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestSnapshotManifestReachesConnectedPeers(t *testing.T) {
	bus := eventbus.New()
	h := &IdenaGossipHandler{bus: bus, peers: newPeerSet()}
	h.subscribeSnapshotManifests()
	peers := []*protoPeer{
		{id: peer.ID("peer-a"), highPriorityRequests: make(chan *request, 1), finished: make(chan struct{})},
		{id: peer.ID("peer-b"), highPriorityRequests: make(chan *request, 1), finished: make(chan struct{})},
	}
	for _, p := range peers {
		require.NoError(t, h.peers.Register(p))
	}
	want := &snapshot.Manifest{CidV2: []byte{1, 2, 3}, Root: common.Hash{4}, Height: 1000}
	bus.Publish(&events.NewSnapshotManifestEvent{Manifest: want})
	for _, p := range peers {
		select {
		case msg := <-p.highPriorityRequests:
			require.Equal(t, uint64(SnapshotManifest), msg.msgcode)
			require.Equal(t, common.MultiShard, msg.shardId)
			data, err := toBytes(msg.msgcode, msg.data)
			require.NoError(t, err)
			var decoded snapshot.Manifest
			require.NoError(t, decoded.FromBytes(data))
			require.Equal(t, want, &decoded)
		default:
			t.Fatalf("connected peer %s did not receive the new manifest", p.id)
		}
	}
}

func TestShouldLogHandshakeFailure(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		peerVersion    string
		want           bool
	}{
		{name: "invalid current build label", currentVersion: "modern-f6db89a", peerVersion: "1.1.2", want: true},
		{name: "invalid peer version", currentVersion: "1.1.2", peerVersion: "development", want: true},
		{name: "newer major", currentVersion: "1.1.2", peerVersion: "2.0.0", want: true},
		{name: "same minor", currentVersion: "1.1.2", peerVersion: "1.1.0", want: true},
		{name: "newer minor", currentVersion: "1.1.2", peerVersion: "1.2.0", want: true},
		{name: "older minor", currentVersion: "1.1.2", peerVersion: "1.0.9", want: false},
		{name: "older major", currentVersion: "2.0.0", peerVersion: "1.9.9", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLogHandshakeFailure(tt.currentVersion, tt.peerVersion); got != tt.want {
				t.Fatalf("shouldLogHandshakeFailure(%q, %q) = %v, want %v", tt.currentVersion, tt.peerVersion, got, tt.want)
			}
		})
	}
}
