package blockchain

import (
	"testing"

	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/core/appstate"
	"github.com/idena-network/idena-go/core/state"
	"github.com/stretchr/testify/require"
)

// requireHeadShardsMatchReadonlyState checks the shard getters against the computation they
// replace: reading the coinbase identity and shards number from a full read-only app state at
// the current head.
func requireHeadShardsMatchReadonlyState(t *testing.T, chain *TestBlockchain, appState *appstate.AppState) {
	readonly, err := appState.Readonly(chain.Head.Height())
	require.NoError(t, err)
	identity := readonly.State.GetIdentity(chain.coinBaseAddress)
	expectedShard := identity.ShiftedShardId()
	expectedModifiedShard := expectedShard
	if identity.State == state.Undefined || identity.State == state.Invite || identity.State == state.Killed {
		expectedModifiedShard = common.MultiShard
	}

	for i := 0; i < 2; i++ {
		shard, err := chain.CoinbaseShard()
		require.NoError(t, err)
		require.Equal(t, expectedShard, shard)
		modifiedShard, err := chain.ModifiedCoinbaseShard()
		require.NoError(t, err)
		require.Equal(t, expectedModifiedShard, modifiedShard)
		require.Equal(t, readonly.State.ShardsNum(), chain.ShardsNum())
	}
}

func TestHeadShardsFollowHead(t *testing.T) {
	chain, appState, _, _ := NewTestBlockchain(true, nil)
	requireHeadShardsMatchReadonlyState(t, chain, appState)

	appState.State.SetShardsNum(4)
	appState.State.SetShardId(chain.coinBaseAddress, 3)
	appState.Commit(nil)
	chain.CommitState()
	requireHeadShardsMatchReadonlyState(t, chain, appState)
	require.Equal(t, uint32(4), chain.ShardsNum())

	chain.GenerateEmptyBlocks(2)
	requireHeadShardsMatchReadonlyState(t, chain, appState)

	appState.State.SetState(chain.coinBaseAddress, state.Killed)
	appState.Commit(nil)
	chain.CommitState()
	requireHeadShardsMatchReadonlyState(t, chain, appState)
	modifiedShard, err := chain.ModifiedCoinbaseShard()
	require.NoError(t, err)
	require.Equal(t, common.MultiShard, modifiedShard)
}
