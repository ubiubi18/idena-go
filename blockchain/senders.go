package blockchain

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/idena-network/idena-go/blockchain/types"
)

const recoverSendersMinTxs = 16

// recoverSenders recovers the senders of a block's transactions on all cores and caches them on
// the transactions, as types.Sender does, before they are validated and applied in order. Sender
// recovery is a pure function of the signed transaction, so this only changes when it is
// computed. Failures are not cached, so an invalid signature fails in sequential validation
// exactly as before.
func recoverSenders(txs []*types.Transaction) {
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
