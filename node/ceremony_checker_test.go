package node

import (
	"testing"

	"github.com/idena-network/idena-go/blockchain"
	"github.com/idena-network/idena-go/core/state"
	"github.com/stretchr/testify/require"
)

func TestCeremonyCheckerFollowsValidationPeriod(t *testing.T) {
	chain, appState, _, _ := blockchain.NewTestBlockchain(true, nil)
	checker := &ceremonyChecker{appState: appState, chain: chain.Blockchain}

	requireMatchesReadonlyState := func(expected bool) {
		readonly, err := appState.Readonly(chain.Head.Height())
		require.NoError(t, err)
		require.Equal(t, readonly.State.ValidationPeriod() >= state.FlipLotteryPeriod, expected)
		for i := 0; i < 2; i++ {
			require.Equal(t, expected, checker.IsRunning())
		}
	}

	requireMatchesReadonlyState(false)
	for _, tc := range []struct {
		period  state.ValidationPeriod
		running bool
	}{
		{state.FlipLotteryPeriod, true},
		{state.ShortSessionPeriod, true},
		{state.LongSessionPeriod, true},
		{state.NonePeriod, false},
	} {
		appState.State.SetValidationPeriod(tc.period)
		appState.Commit(nil)
		chain.CommitState()
		require.Equal(t, tc.period, appState.State.ValidationPeriod())
		requireMatchesReadonlyState(tc.running)
	}
}
