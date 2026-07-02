//go:build !idena_memory_ipfs

package events

import (
	"time"

	"github.com/idena-network/idena-go/common/eventbus"
	iface "github.com/ipfs/kubo/core/coreiface"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core"
)

type IpfsPortChangedEvent struct {
	Host   core.Host
	PubSub *pubsub.PubSub
}

func (i IpfsPortChangedEvent) EventID() eventbus.EventID {
	return IpfsPortChangedEventId
}

type PeersEvent struct {
	PeersData []iface.ConnectionInfo
	Time      time.Time
}

func (e *PeersEvent) EventID() eventbus.EventID {
	return PeersEventID
}
