package mempool

import (
	"fmt"
	"github.com/asaskevich/EventBus"
	"github.com/google/tink/go/subtle/random"
	"gx/ipfs/QmPVkJMTeRC6iBByPWdrRkD3BE5UXsj5HPzb4kPqL186mS/testify/require"
	"idena-go/blockchain/types"
	"idena-go/common"
	"testing"
)

func getRandAddr() common.Address {
	addr := common.Address{}
	addr.SetBytes(random.GetRandomBytes(20))
	return addr
}

func TestEvidenceMap_CalculateBitmap(t *testing.T) {
	require := require.New(t)

	bus := EventBus.New()

	em := NewEvidenceMap(bus)

	const candidatesCount = 10000
	const skipCandidate = 5
	var addrs []common.Address

	for i := 0; i < candidatesCount; i++ {
		addr := getRandAddr()
		addrs = append(addrs, addr)
		if i != skipCandidate {
			em.newTx(&types.Transaction{
				To:      &addr,
				Payload: random.GetRandomBytes(common.HashLength),
				Type:    types.SubmitAnswers,
			})
		}
	}

	ignored := []common.Address{}

	for i := 0; i < candidatesCount; i++ {
		if i%2 == 0 {
			ignored = append(ignored, addrs[i])
		}
	}
	bytesArray := em.CalculateBitmap(addrs, ignored)

	fmt.Printf("size of bitmap for %v candidates is %v bytes\n", candidatesCount, len(bytesArray))

	rmap := common.NewBitmap(candidatesCount)
	rmap.Read(bytesArray)

	require.True(rmap.Contains(1))
	require.False(rmap.Contains(skipCandidate))

	for i := 0; i < candidatesCount; i++ {
		if i%2 == 0 {
			require.False(rmap.Contains(uint32(i)))
		} else if i != skipCandidate {
			require.True(rmap.Contains(uint32(i)))
		}
	}

}
