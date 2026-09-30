package blockchain

import (
	"sync"

	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/core/state"
)

type headShards struct {
	head          *types.Header
	height        uint64
	shardsNum     uint32
	coinbaseShard common.ShardId
	coinbaseState state.IdentityState
}

// headShardsCache holds the shard values of the state at the current head. The gossip handler
// reads them on every new block and in its peer management loops. They are read once per head,
// from the state tree alone, instead of building a read-only app state (which clones the
// validators cache) on every call. Any change of head, including a switch to another block at
// the same height, refreshes them.
type headShardsCache struct {
	mutex  sync.Mutex
	valid  bool
	values headShards
}

func (chain *Blockchain) headShardsInfo() (headShards, error) {
	head := chain.Head
	height := head.Height()
	cache := &chain.headShards
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if cache.valid && cache.values.head == head && cache.values.height == height {
		return cache.values, nil
	}
	stateDb, err := chain.appState.State.Readonly(int64(height))
	if err != nil {
		return headShards{}, err
	}
	identity := stateDb.GetIdentity(chain.coinBaseAddress)
	cache.values = headShards{
		head:          head,
		height:        height,
		shardsNum:     stateDb.ShardsNum(),
		coinbaseShard: identity.ShiftedShardId(),
		coinbaseState: identity.State,
	}
	cache.valid = true
	return cache.values, nil
}
