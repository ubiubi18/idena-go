package vm

import (
	"math/big"
	"testing"

	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/eventbus"
	"github.com/idena-network/idena-go/config"
	"github.com/idena-network/idena-go/core/appstate"
	"github.com/idena-network/idena-go/crypto"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"
)

func TestVmImplRunMalformedDeployAttachment(t *testing.T) {
	db := dbm.NewMemDB()
	appState, err := appstate.NewAppState(db, eventbus.New())
	require.NoError(t, err)
	appState.Initialize(0)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)

	tx, err := types.SignTx(&types.Transaction{
		Type:    types.DeployContractTx,
		Amount:  big.NewInt(0),
		MaxFee:  big.NewInt(0),
		Payload: []byte{0xff},
	}, key)
	require.NoError(t, err)

	cfg := &config.Config{
		Consensus:        config.GetDefaultConsensusConfig(),
		Blockchain:       &config.BlockchainConfig{},
		OfflineDetection: config.GetDefaultOfflineDetectionConfig(),
		Mempool:          config.GetDefaultMempoolConfig(),
	}
	header := &types.Header{ProposedHeader: &types.ProposedHeader{Height: 1}}
	receipt := NewVmImpl(appState, nil, header, nil, cfg).Run(tx, &from, 1000, true)

	require.NotNil(t, receipt)
	require.False(t, receipt.Success)
	require.EqualError(t, receipt.Error, "can't parse attachment")
	require.Equal(t, common.Address{}, receipt.ContractAddress)
}
