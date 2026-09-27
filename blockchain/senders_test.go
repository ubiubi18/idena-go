package blockchain

import (
	"math/big"
	"sync"
	"testing"

	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/crypto"
	"github.com/stretchr/testify/require"
)

func decodedTx(t *testing.T, tx *types.Transaction) *types.Transaction {
	data, err := tx.ToBytes()
	require.NoError(t, err)
	cp := new(types.Transaction)
	require.NoError(t, cp.FromBytes(data))
	return cp
}

// sendersTestTxs returns decoded transactions with valid signatures, signatures that recover
// another key and signatures that cannot be recovered at all.
func sendersTestTxs(t *testing.T, n int) []*types.Transaction {
	var txs []*types.Transaction
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateKey()
		txType := types.SendTx
		if i%3 == 1 {
			txType = types.SubmitLongAnswersTx
		}
		tx, err := types.SignTx(&types.Transaction{AccountNonce: uint32(i), Type: txType, Amount: big.NewInt(int64(i)), Payload: []byte{byte(i)}}, key)
		require.NoError(t, err)
		switch i % 5 {
		case 3:
			tx.Signature[5] ^= 0xff // recovers another key, or nothing
		case 4:
			tx.Signature = tx.Signature[:10] // malformed
		}
		txs = append(txs, decodedTx(t, tx))
	}
	return txs
}

// recoverSenders must leave every transaction with exactly the sender that sequential recovery
// from a fresh copy produces, and must not cache failures.
func TestRecoverSendersMatchesSequentialRecovery(t *testing.T) {
	for _, n := range []int{recoverSendersMinTxs, 64} {
		txs := sendersTestTxs(t, n)
		blockTxs := make([]*types.Transaction, len(txs))
		for i, tx := range txs {
			blockTxs[i] = decodedTx(t, tx)
		}

		recoverSenders(blockTxs)

		for i, tx := range blockTxs {
			expectedSender, expectedErr := types.Sender(decodedTx(t, txs[i]))
			if expectedErr != nil {
				_, err := types.Sender(tx)
				require.Equal(t, expectedErr, err, "tx %d", i)
				continue
			}
			// Replace the signature: only a sender cached by recoverSenders can still match.
			tx.Signature = []byte{1, 2, 3}
			sender, err := types.Sender(tx)
			require.NoError(t, err, "tx %d", i)
			require.Equal(t, expectedSender, sender, "tx %d", i)
		}
	}
}

// Other goroutines, such as the mempool, may recover the sender of the same transaction objects
// while a block is validated.
func TestRecoverSendersConcurrentWithSender(t *testing.T) {
	txs := sendersTestTxs(t, 64)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, tx := range txs {
			types.Sender(tx)
		}
	}()
	recoverSenders(txs)
	wg.Wait()
}
