package blockchain

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/idena-network/idena-go/blockchain/types"
)

const recoverSendersMinTxs = 16
const recoverSendersBatchSize = 64

// recoverSenders caches senders for at most one batch before sequential validation. Bounding
// the batch limits wasted recovery when validation rejects an early transaction. Failures are
// not cached, so invalid signatures still fail in sequential validation.
func recoverSenders(txs []*types.Transaction) {
	if len(txs) > recoverSendersBatchSize {
		txs = txs[:recoverSendersBatchSize]
	}
	if len(txs) < recoverSendersMinTxs {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > len(txs) {
		workers = len(txs)
	}
	var next int64 = -1
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for {
				i := atomic.AddInt64(&next, 1)
				if i >= int64(len(txs)) {
					return
				}
				types.Sender(txs[i])
			}
		}()
	}
	wg.Wait()
}
