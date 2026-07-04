package deferredtx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/idena-network/idena-go/blockchain"
	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/blockchain/validation"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/config"
	"github.com/idena-network/idena-go/core/appstate"
	"github.com/idena-network/idena-go/stats/collector"
	"github.com/idena-network/idena-go/vm"
	"github.com/idena-network/idena-go/vm/embedded"
	"github.com/idena-network/idena-go/vm/wasm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var fakeVmError error

type fakeVm struct {
}

func (f fakeVm) ContractAddr(tx *types.Transaction, from *common.Address) common.Address {
	panic("implement me")
}

func (f fakeVm) IsWasm(tx *types.Transaction) bool {
	return false
}

func (f fakeVm) Read(contractAddr common.Address, method string, args ...[]byte) ([]byte, error) {
	panic("implement me")
}

func (f fakeVm) Run(tx *types.Transaction, from *common.Address, gasLimit int64, commitToEnv bool) *types.TxReceipt {
	return &types.TxReceipt{
		Error:   fakeVmError,
		Success: fakeVmError == nil,
	}
}

type fakeTxPool struct {
	counter int
}

func (f *fakeTxPool) GetPriorityTransaction() []*types.Transaction {
	panic("implement me")
}

func (f *fakeTxPool) AddInternalTx(tx *types.Transaction) error {
	f.counter++
	return nil
}
func (f *fakeTxPool) AddExternalTxs(txType validation.TxType, txs ...*types.Transaction) error {
	panic("implement me")
}

func (f fakeTxPool) GetPendingTransaction(bool, bool, common.ShardId, bool) []*types.Transaction {
	panic("implement me")
}

func (f fakeTxPool) IsSyncing() bool {
	return false
}

func TestJobPersistReturnsOpenFileError(t *testing.T) {
	datadir := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(datadir, []byte("file"), 0600))

	job := &Job{
		datadir: datadir,
		txs:     new(DeferredTxs),
	}

	require.Error(t, job.persist())
}

func TestJobOpenFileCreatesPrivateStorage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX file mode bits")
	}

	datadir := t.TempDir()
	job := &Job{datadir: datadir}

	file, err := job.openFile()
	require.NoError(t, err)
	require.NoError(t, file.Close())

	assertMode(t, filepath.Join(datadir, Folder), 0700)
	assertMode(t, filepath.Join(datadir, Folder, "txs"), 0600)
}

func assertMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, mode, info.Mode().Perm())
}

func TestJob_tryLater(t *testing.T) {

	fakeVmError = embedded.NewContractError("", true)
	chain, appState, _, _ := blockchain.NewTestBlockchain(false, nil)
	defer chain.SecStore().Destroy()
	datadir := t.TempDir()

	txPool := &fakeTxPool{}
	job, _ := NewJob(chain.Bus(), datadir, appState, chain.Blockchain, txPool, nil, chain.SecStore(), func(appState *appstate.AppState, blockHeaderProvider wasm.BlockHeaderProvider, block *types.Header, statsCollector collector.StatsCollector, cfg *config.Config) vm.VM {
		return &fakeVm{}
	})
	coinbase := chain.SecStore().GetAddress()

	require.NoError(t, job.AddDeferredTx(coinbase, &common.Address{0x1}, common.DnaBase, nil, nil, 10))
	require.NoError(t, job.AddDeferredTx(coinbase, &common.Address{0x1}, common.DnaBase, nil, nil, 50))
	require.Len(t, job.txs.Txs, 2)

	chain.GenerateEmptyBlocks(8)
	require.Len(t, job.txs.Txs, 2)

	chain.GenerateEmptyBlocks(1)
	require.Equal(t, 1, job.txs.Txs[0].sendTry)
	require.Equal(t, uint64(11), job.txs.Txs[0].BroadcastBlock)

	chain.GenerateEmptyBlocks(1)
	require.Equal(t, 2, job.txs.Txs[0].sendTry)
	require.Equal(t, uint64(13), job.txs.Txs[0].BroadcastBlock)

	chain.GenerateEmptyBlocks(4)
	require.Equal(t, 3, job.txs.Txs[0].sendTry)
	require.Equal(t, uint64(17), job.txs.Txs[0].BroadcastBlock)

	chain.GenerateEmptyBlocks(8)
	require.Equal(t, 4, job.txs.Txs[0].sendTry)
	require.Equal(t, uint64(25), job.txs.Txs[0].BroadcastBlock)

	chain.GenerateEmptyBlocks(2)
	require.Equal(t, 5, job.txs.Txs[0].sendTry)
	require.Equal(t, uint64(33), job.txs.Txs[0].BroadcastBlock)

	fakeVmError = nil
	chain.GenerateEmptyBlocks(10)

	require.Len(t, job.txs.Txs, 1)
	require.Equal(t, 1, txPool.counter)

	fakeVmError = errors.New("custom error")
	chain.GenerateEmptyBlocks(50)

	require.Len(t, job.txs.Txs, 0)
	require.Equal(t, 1, txPool.counter)
}
