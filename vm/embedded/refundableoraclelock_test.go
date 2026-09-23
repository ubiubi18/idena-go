package embedded

import (
	"crypto/ecdsa"
	"github.com/idena-network/idena-go/blockchain/attachments"
	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/eventbus"
	math2 "github.com/idena-network/idena-go/common/math"
	"github.com/idena-network/idena-go/core/appstate"
	"github.com/idena-network/idena-go/core/state"
	"github.com/idena-network/idena-go/crypto"
	"github.com/idena-network/idena-go/vm/env"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

func TestNormalizeOracleVotingFee(t *testing.T) {
	tests := []struct {
		fee      uint64
		expected uint64
	}{
		{fee: 0, expected: 0},
		{fee: 99999, expected: 99999},
		{fee: 100000, expected: 100000},
		{fee: 100001, expected: 100000},
		{fee: math.MaxInt64, expected: 100000},
		{fee: math.MaxInt64 + 1, expected: 0},
		{fee: ^uint64(0) - 99999, expected: 0},
		{fee: ^uint64(0), expected: 0},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, normalizeOracleVotingFee(test.fee), "fee %d", test.fee)
	}
}

// The network computes the fee with the upstream expression on 64-bit nodes.
func TestNormalizeOracleVotingFeeMatchesUpstream64Bit(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("the upstream expression depends on a 64-bit int")
	}
	upstream := func(fee uint64) uint64 {
		return uint64(math2.MaxInt(0, math2.MinInt(100000, int(fee))))
	}

	fees := []uint64{0, 1, 99999, 100000, 100001, math.MaxInt32, math.MaxUint32,
		math.MaxInt64 - 1, math.MaxInt64, math.MaxInt64 + 1, math.MaxInt64 + 100000,
		^uint64(0) - 100000, ^uint64(0) - 1, ^uint64(0)}
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < 100000; i++ {
		fees = append(fees, rnd.Uint64(), rnd.Uint64()>>uint(rnd.Intn(64)))
	}
	for _, fee := range fees {
		require.Equal(t, upstream(fee), normalizeOracleVotingFee(fee), "fee %d", fee)
	}
}

func TestRefundableEvidenceLock_Call(t *testing.T) {

	oracleVotingAddr := common.Address{0x1}
	successAddr := common.Address{0x2}
	failAddr := common.Address{0x3}

	db := dbm.NewMemDB()
	appState, _ := appstate.NewAppState(db, eventbus.New())

	appState.State.DeployContract(oracleVotingAddr, OracleVotingContract, common.DnaBase)

	appState.State.SetFeePerGas(big.NewInt(1))
	rnd := rand.New(rand.NewSource(1))
	key, _ := crypto.GenerateKeyFromSeed(rnd)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	appState.State.SetState(addr, state.Newbie)
	appState.State.SetBalance(addr, common.DnaBase)
	appState.State.SetPubKey(addr, crypto.FromECDSAPub(&key.PublicKey))
	appState.IdentityState.SetValidated(addr, true)

	var identities []*ecdsa.PrivateKey

	for i := 0; i < 2000; i++ {
		key, _ := crypto.GenerateKeyFromSeed(rnd)
		identities = append(identities, key)
		addr := crypto.PubkeyToAddress(key.PublicKey)
		appState.State.SetState(addr, state.Newbie)
		appState.State.SetBalance(addr, common.DnaBase)
		appState.State.SetPubKey(addr, crypto.FromECDSAPub(&key.PublicKey))
		appState.IdentityState.SetValidated(addr, true)
	}
	appState.Commit(nil)

	appState.Initialize(1)

	attachment := attachments.CreateDeployContractAttachment(RefundableOracleLockContract, nil, nil, oracleVotingAddr.Bytes(),
		common.ToBytes(byte(1)), successAddr.Bytes(), failAddr.Bytes(), nil, common.ToBytes(uint64(1000)), common.ToBytes(uint64(5000)))
	payload, err := attachment.ToBytes()
	require.NoError(t, err)

	tx := &types.Transaction{
		Epoch:        0,
		AccountNonce: 1,
		Type:         types.DeployContractTx,
		Amount:       common.DnaBase,
		Payload:      payload,
	}
	tx, _ = types.SignTx(tx, key)
	ctx := env.NewDeployContextImpl(tx, nil, attachment.CodeHash)

	gas := new(env.GasCounter)
	gas.Reset(-1)

	// deploy
	e := env.NewEnvImp(appState, createHeader(2, 1), gas, nil)
	contract := NewRefundableOracleLock2(ctx, e, nil)

	contractAddr := ctx.ContractAddr()

	appState.State.AddBalance(contractAddr, big.NewInt(0).Mul(common.DnaBase, big.NewInt(500)))
	require.NoError(t, contract.Deploy(attachment.Args...))
	e.Commit()

	for i := 0; i < 100; i++ {
		key := identities[i]

		attachment := attachments.CreateCallContractAttachment("deposit")
		payload, err := attachment.ToBytes()
		require.NoError(t, err)

		tx := &types.Transaction{
			Epoch:        0,
			AccountNonce: 1,
			Type:         types.CallContractTx,
			To:           &contractAddr,
			Amount:       common.DnaBase,
			Payload:      payload,
		}
		tx, _ = types.SignTx(tx, key)

		ctx := env.NewCallContextImpl(tx, nil, RefundableOracleLockContract)
		gas.Reset(-1)
		e = env.NewEnvImp(appState, createHeader(4, 21), gas, nil)
		contract = NewRefundableOracleLock2(ctx, e, nil)
		err = contract.Call(attachment.Method, attachment.Args...)
		e.Commit()
		require.NoError(t, err)
	}

	require.True(t, big.NewInt(0).Mul(common.DnaBase, big.NewInt(5)).Cmp(appState.State.GetBalance(oracleVotingAddr)) == 0)

	totalSum := e.ReadContractData(contractAddr, []byte("sum"))

	require.Equal(t, totalSum, big.NewInt(0).Mul(big.NewInt(100), common.DnaBase).Bytes())

	deposit := e.ReadContractData(contractAddr, append([]byte("deposits"), crypto.PubkeyToAddress(identities[50].PublicKey).Bytes()...))

	require.Equal(t, common.DnaBase.Bytes(), deposit)

	callAttach := attachments.CreateCallContractAttachment("refund")
	payload, _ = attachment.ToBytes()

	tx = &types.Transaction{
		Epoch:        0,
		AccountNonce: 1,
		Type:         types.CallContractTx,
		To:           &contractAddr,
		Amount:       common.DnaBase,
		Payload:      payload,
	}
	tx, _ = types.SignTx(tx, key)
	gas.Reset(-1)
	e = env.NewEnvImp(appState, createHeader(4, 21), gas, nil)
	contract = NewRefundableOracleLock2(env.NewCallContextImpl(tx, nil, RefundableOracleLockContract), e, nil)
	err = contract.Call(callAttach.Method, callAttach.Args...)
	require.Error(t, err)
	e.Reset()

	callAttach = attachments.CreateCallContractAttachment("push")
	payload, _ = attachment.ToBytes()

	tx = &types.Transaction{
		Epoch:        0,
		AccountNonce: 1,
		Type:         types.CallContractTx,
		To:           &contractAddr,
		Amount:       common.DnaBase,
		Payload:      payload,
	}
	tx, _ = types.SignTx(tx, key)
	gas.Reset(-1)
	e = env.NewEnvImp(appState, createHeader(4, 21), gas, nil)
	contract = NewRefundableOracleLock2(env.NewCallContextImpl(tx, nil, RefundableOracleLockContract), e, nil)
	err = contract.Call(callAttach.Method, callAttach.Args...)
	require.Error(t, err)
	e.Reset()

}
