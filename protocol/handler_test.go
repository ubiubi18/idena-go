package protocol

import (
	"errors"
	"testing"

	core "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pprotocol "github.com/libp2p/go-libp2p/core/protocol"
)

type recordingNetwork struct {
	network.Network
	peers      []peer.ID
	closed     []peer.ID
	closeError map[peer.ID]error
}

func (n *recordingNetwork) Peers() []peer.ID {
	return append([]peer.ID(nil), n.peers...)
}

func (n *recordingNetwork) ClosePeer(id peer.ID) error {
	n.closed = append(n.closed, id)
	return n.closeError[id]
}

type recordingHost struct {
	core.Host
	network    *recordingNetwork
	protocol   libp2pprotocol.ID
	matcher    func(libp2pprotocol.ID) bool
	streamFunc network.StreamHandler
}

func (h *recordingHost) Network() network.Network {
	return h.network
}

func (h *recordingHost) SetStreamHandlerMatch(id libp2pprotocol.ID, matcher func(libp2pprotocol.ID) bool, handler network.StreamHandler) {
	h.protocol = id
	h.matcher = matcher
	h.streamFunc = handler
}

func TestRegisterIdenaStreamHandlerRefreshesPreexistingPeers(t *testing.T) {
	peerA := peer.ID("peer-a")
	peerB := peer.ID("peer-b")
	wantError := errors.New("close failed")
	networkRecorder := &recordingNetwork{
		peers:      []peer.ID{peerA, peerB},
		closeError: map[peer.ID]error{peerB: wantError},
	}
	host := &recordingHost{network: networkRecorder}
	handler := func(network.Stream) {}

	closed, failed := registerIdenaStreamHandler(host, handler)

	if closed != 1 || failed != 1 {
		t.Fatalf("registerIdenaStreamHandler() = (%d, %d), want (1, 1)", closed, failed)
	}
	if host.protocol != IdenaProtocol || host.streamFunc == nil {
		t.Fatal("Idena stream handler was not registered")
	}
	if !host.matcher(IdenaProtocol) || host.matcher("/idena/gossip/2.0.0") {
		t.Fatal("registered matcher does not preserve Idena protocol compatibility")
	}
	if len(networkRecorder.closed) != 2 || networkRecorder.closed[0] != peerA || networkRecorder.closed[1] != peerB {
		t.Fatalf("closed peers = %v, want [%s %s]", networkRecorder.closed, peerA, peerB)
	}
}
