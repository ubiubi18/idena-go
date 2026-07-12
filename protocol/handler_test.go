package protocol

import (
	"context"
	"errors"
	"testing"

	core "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
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
	peerstore  *recordingPeerstore
	protocol   libp2pprotocol.ID
	matcher    func(libp2pprotocol.ID) bool
	streamFunc network.StreamHandler
	connected  []peer.AddrInfo
}

type recordingPeerstore struct {
	peerstore.Peerstore
	peerInfo map[peer.ID]peer.AddrInfo
}

func (p *recordingPeerstore) PeerInfo(id peer.ID) peer.AddrInfo {
	return p.peerInfo[id]
}

func (h *recordingHost) Network() network.Network {
	return h.network
}

func (h *recordingHost) Peerstore() peerstore.Peerstore {
	return h.peerstore
}

func (h *recordingHost) Connect(_ context.Context, peerInfo peer.AddrInfo) error {
	h.connected = append(h.connected, peerInfo)
	return nil
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
	host := &recordingHost{
		network: networkRecorder,
		peerstore: &recordingPeerstore{peerInfo: map[peer.ID]peer.AddrInfo{
			peerA: {ID: peerA},
			peerB: {ID: peerB},
		}},
	}
	handler := func(network.Stream) {}

	reconnectTargets, closed, failed := registerIdenaStreamHandler(host, handler)

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
	if len(reconnectTargets) != 1 || reconnectTargets[0].ID != peerA {
		t.Fatalf("reconnect targets = %v, want peer %s", reconnectTargets, peerA)
	}

	reconnected, reconnectFailed := reconnectIdenaPeers(host, reconnectTargets)
	if reconnected != 1 || reconnectFailed != 0 {
		t.Fatalf("reconnectIdenaPeers() = (%d, %d), want (1, 0)", reconnected, reconnectFailed)
	}
	if len(host.connected) != 1 || host.connected[0].ID != peerA {
		t.Fatalf("connected peers = %v, want peer %s", host.connected, peerA)
	}
}
