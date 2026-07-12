package protocol

import (
	"context"
	"testing"

	core "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	libp2pprotocol "github.com/libp2p/go-libp2p/core/protocol"
)

type protocolStore struct {
	peerstore.Peerstore
	protocols []libp2pprotocol.ID
}

func (s *protocolStore) GetProtocols(peer.ID) ([]libp2pprotocol.ID, error) {
	return append([]libp2pprotocol.ID(nil), s.protocols...), nil
}

type streamOpeningHost struct {
	core.Host
	store           peerstore.Peerstore
	stream          network.Stream
	newStreamCalls  int
	openedPeer      peer.ID
	openedProtocols []libp2pprotocol.ID
}

func (h *streamOpeningHost) Peerstore() peerstore.Peerstore {
	return h.store
}

func (h *streamOpeningHost) NewStream(_ context.Context, id peer.ID, protocols ...libp2pprotocol.ID) (network.Stream, error) {
	h.newStreamCalls++
	h.openedPeer = id
	h.openedProtocols = append([]libp2pprotocol.ID(nil), protocols...)
	return h.stream, nil
}

type existingStreamConn struct {
	network.Conn
	remote          peer.ID
	getStreamsCalls int
}

func (c *existingStreamConn) RemotePeer() peer.ID {
	return c.remote
}

func (c *existingStreamConn) GetStreams() []network.Stream {
	c.getStreamsCalls++
	return nil
}

type sentinelStream struct {
	network.Stream
}

func TestFindOrOpenStreamAlwaysNegotiatesANewStream(t *testing.T) {
	remote := peer.ID("remote-peer")
	wantStream := &sentinelStream{}
	host := &streamOpeningHost{
		store:  &protocolStore{protocols: []libp2pprotocol.ID{IdenaProtocol}},
		stream: wantStream,
	}
	conn := &existingStreamConn{remote: remote}
	manager := &ConnManager{host: host}

	got, err := manager.findOrOpenStream(conn)
	if err != nil {
		t.Fatalf("findOrOpenStream() error = %v", err)
	}
	if got != wantStream {
		t.Fatal("findOrOpenStream() did not return the newly negotiated stream")
	}
	if conn.getStreamsCalls != 0 {
		t.Fatalf("Conn.GetStreams called %d times; existing raw streams must not be reused", conn.getStreamsCalls)
	}
	if host.newStreamCalls != 1 || host.openedPeer != remote {
		t.Fatalf("Host.NewStream calls = %d, peer = %q", host.newStreamCalls, host.openedPeer)
	}
	if len(host.openedProtocols) != 1 || host.openedProtocols[0] != IdenaProtocol {
		t.Fatalf("negotiated protocols = %v, want [%s]", host.openedProtocols, IdenaProtocol)
	}
}
