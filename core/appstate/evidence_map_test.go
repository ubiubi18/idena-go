package appstate

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/google/tink/go/subtle/random"
	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/eventbus"
	"github.com/idena-network/idena-go/crypto"
	"github.com/idena-network/idena-go/tests"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEvidenceMap_CalculateBitmap(t *testing.T) {
	require := require.New(t)

	bus := eventbus.New()
	em := NewEvidenceMap(bus)
	now := time.Now().UTC().Add(-24 * time.Second)
	em.SetShortSessionTime(now, time.Minute*1)

	const candidatesCount = 10000
	const txCandidate = 1
	const delayedKeyAuthor = 10
	const delayedKeyPackageAuthor = 100
	var addrs []common.Address

	for i := 0; i < candidatesCount; i++ {
		addr := tests.GetRandAddr()

		if i == txCandidate {
			key, _ := crypto.GenerateKey()
			a := crypto.PubkeyToAddress(key.PublicKey)
			tx := &types.Transaction{
				Payload: random.GetRandomBytes(common.HashLength),
				Type:    types.SubmitAnswersHashTx,
			}
			signed, _ := types.SignTx(tx, key)
			em.newTx(signed)
			addrs = append(addrs, a)
		} else {
			addrs = append(addrs, addr)
		}
	}

	additional := []common.Address{}

	for i := 0; i < candidatesCount; i++ {
		if i%2 == 0 {
			additional = append(additional, addrs[i])
		}
	}

	em.NewFlipsKey(addrs[txCandidate])
	em.NewFlipKeyPackage(addrs[txCandidate])
	time.Sleep(2 * time.Second)
	em.NewFlipsKey(addrs[delayedKeyAuthor])
	em.NewFlipKeyPackage(addrs[delayedKeyPackageAuthor])

	m := em.CalculateBitmap(addrs, additional, func(a common.Address) uint8 {
		if a == addrs[delayedKeyAuthor] || a == addrs[txCandidate] || a == addrs[delayedKeyPackageAuthor] {
			return 1
		}
		return 0
	})

	buf := new(bytes.Buffer)

	m.WriteTo(buf)
	bytesArray := buf.Bytes()
	fmt.Printf("size of bitmap for %v candidates is %v bytes\n", candidatesCount, len(bytesArray))

	rmap := common.NewBitmap(candidatesCount)
	require.NoError(rmap.Read(bytesArray))

	require.True(rmap.Contains(txCandidate))
	require.False(rmap.Contains(delayedKeyAuthor))
	require.False(rmap.Contains(delayedKeyPackageAuthor))

	for i := 0; i < candidatesCount; i++ {
		if i%2 == 0 && i != delayedKeyAuthor && i != delayedKeyPackageAuthor {
			require.True(rmap.Contains(uint32(i)))
		} else if i != txCandidate {
			require.False(rmap.Contains(uint32(i)))
		}
	}
}

func TestEvidenceMap_CalculateApprovedCandidates(t *testing.T) {
	require := require.New(t)

	bus := eventbus.New()
	em := NewEvidenceMap(bus)

	const candidatesCount = 3
	var candidates []common.Address

	for i := 0; i < candidatesCount; i++ {
		addr := tests.GetRandAddr()
		candidates = append(candidates, addr)
	}
	// first map
	var maps [][]byte
	m := common.NewBitmap(candidatesCount)
	m.Add(0)
	m.Add(1)
	m.Add(2)

	buf := new(bytes.Buffer)
	m.WriteTo(buf)
	maps = append(maps, buf.Bytes())
	maps = append(maps, nil)

	// second map
	m = common.NewBitmap(candidatesCount)
	m.Add(0)
	m.Add(2)

	buf = new(bytes.Buffer)
	m.WriteTo(buf)
	maps = append(maps, buf.Bytes())

	// third map
	m = common.NewBitmap(candidatesCount)
	m.Add(0)

	buf = new(bytes.Buffer)
	m.WriteTo(buf)
	maps = append(maps, buf.Bytes())

	// act
	appproved := em.CalculateApprovedCandidates(candidates, maps)

	// assert
	require.Len(appproved, 1)

	require.Contains(appproved, candidates[0])

}

func TestEvidenceMap_CalculateApprovedCandidatesLegacyMaps(t *testing.T) {
	candidate := common.Address{1}
	duplicateVote, err := hex.DecodeString("013a300000020000000000000000000000180000001a00000000000000")
	require.NoError(t, err)

	tests := []struct {
		name     string
		maps     [][]byte
		approved bool
	}{
		{name: "unknown prefix counts as empty vote", maps: [][]byte{{0x02, 0x01}, {0xff, 0x00}}},
		{name: "truncated roaring counts as empty vote", maps: [][]byte{{0x02, 0x01}, {0x01}}},
		{name: "empty map cannot lower majority", maps: [][]byte{{0x02, 0x01}, nil}},
		{name: "unknown prefix contributes vote", maps: [][]byte{{0x02, 0x01}, {0xff, 0x01}, {0x02}}, approved: true},
		{name: "high bits outside candidate range are ignored", maps: [][]byte{{0x02, 0x01}, {0x02, 0x03}, {0x02}}, approved: true},
		{name: "duplicate roaring values remain separate votes", maps: [][]byte{duplicateVote, {0x02}, {0x02}}, approved: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (&EvidenceMap{}).CalculateApprovedCandidates([]common.Address{candidate}, tt.maps)
			require.Equal(t, tt.approved, len(got) == 1)
		})
	}
}

func TestEvidenceMap_CalculateApprovedCandidatesLegacyCardinality(t *testing.T) {
	// The legacy Roaring reader trusts the declared bitmap cardinality when
	// allocating ToArray, leaving zero-valued votes after the actual set bits.
	encoded := make([]byte, 1+16+8192)
	encoded[0] = 0x01
	binary.LittleEndian.PutUint32(encoded[1:], 12346)
	binary.LittleEndian.PutUint32(encoded[5:], 1)
	binary.LittleEndian.PutUint16(encoded[11:], 4096) // Declares 4097 values.
	binary.LittleEndian.PutUint32(encoded[13:], 16)
	encoded[17] = 0x02 // Only candidate 1 is actually set.

	candidates := []common.Address{{1}, {2}}
	got := (&EvidenceMap{}).CalculateApprovedCandidates(candidates, [][]byte{encoded})
	require.Equal(t, candidates, got)
}
