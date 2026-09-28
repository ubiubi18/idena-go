package api

import (
	"testing"

	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/eventbus"
	"github.com/idena-network/idena-go/common/hexutil"
	"github.com/idena-network/idena-go/core/appstate"
	"github.com/idena-network/idena-go/core/state"
	"github.com/stretchr/testify/require"
	db2 "github.com/tendermint/tm-db"
)

// oldBatchReadPerKey reproduces the original BatchReadData behavior, which
// resolved the read-only state once per key inside the loop. With a fixed
// snapshot (as in these tests) it must produce exactly what the hoisted
// batchReadContractData produces, so the refactor stays behavior-preserving.
func oldBatchReadPerKey(stateDb *state.StateDB, contract common.Address, keys []KeyWithFormat) []ContractData {
	res := make([]ContractData, 0, len(keys))
	for _, keyWithFormat := range keys {
		data := ContractData{Key: keyWithFormat.Key}
		if value := stateDb.GetContractValue(contract, []byte(keyWithFormat.Key)); value != nil {
			var err error
			data.Value, err = conversion(keyWithFormat.Format, value)
			if err != nil {
				data.Error = err.Error()
			}
		} else {
			data.Error = "data is nil"
		}
		res = append(res, data)
	}
	return res
}

func newTestAppState(t *testing.T) *appstate.AppState {
	t.Helper()
	appState, err := appstate.NewAppState(db2.NewMemDB(), eventbus.New())
	require.NoError(t, err)
	return appState
}

func TestBatchReadContractData_ValuesFormatsAndMissing(t *testing.T) {
	appState := newTestAppState(t)
	contract := common.Address{0x1}

	appState.State.SetContractValue(contract, []byte("s"), []byte("hello"))
	appState.State.SetContractValue(contract, []byte("h"), []byte{0xde, 0xad})

	keys := []KeyWithFormat{
		{Key: "s", Format: "string"},
		{Key: "h", Format: "hex"},
		{Key: "missing", Format: "string"},
	}

	got := batchReadContractData(appState.State, contract, keys)

	require.Len(t, got, len(keys))
	// order preserved, per-key key echoed back
	require.Equal(t, "s", got[0].Key)
	require.Equal(t, "h", got[1].Key)
	require.Equal(t, "missing", got[2].Key)
	// present keys converted per format, no error
	require.Equal(t, "hello", got[0].Value)
	require.Empty(t, got[0].Error)
	require.Equal(t, hexutil.Encode([]byte{0xde, 0xad}), got[1].Value)
	require.Empty(t, got[1].Error)
	// missing key: nil value, "data is nil" error
	require.Nil(t, got[2].Value)
	require.Equal(t, "data is nil", got[2].Error)
}

// TestBatchReadContractData_MatchesPerKeyReads is the regression guard for the
// hoist: reading every key from a single snapshot must equal the original
// per-key reads for the same state.
func TestBatchReadContractData_MatchesPerKeyReads(t *testing.T) {
	appState := newTestAppState(t)
	contract := common.Address{0x2}
	other := common.Address{0x3} // different contract, must not leak in

	appState.State.SetContractValue(contract, []byte("a"), []byte("1"))
	appState.State.SetContractValue(contract, []byte("b"), []byte{0x02})
	appState.State.SetContractValue(contract, []byte("c"), []byte("three"))
	appState.State.SetContractValue(other, []byte("a"), []byte("wrong-contract"))

	keys := []KeyWithFormat{
		{Key: "a", Format: "string"},
		{Key: "b", Format: "hex"},
		{Key: "c", Format: "string"},
		{Key: "a", Format: "hex"},   // same key, different format
		{Key: "zzz", Format: "hex"}, // missing
		{Key: "", Format: "string"}, // empty key, missing
	}

	got := batchReadContractData(appState.State, contract, keys)
	want := oldBatchReadPerKey(appState.State, contract, keys)
	require.Equal(t, want, got)

	// contract isolation: value under `other` never appears.
	require.Equal(t, "1", got[0].Value)
	require.Equal(t, hexutil.Encode([]byte("1")), got[3].Value)
}

func TestBatchReadContractData_Empty(t *testing.T) {
	appState := newTestAppState(t)
	require.Empty(t, batchReadContractData(appState.State, common.Address{0x1}, nil))
}
