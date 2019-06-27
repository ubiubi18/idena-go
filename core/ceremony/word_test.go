package ceremony

import (
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/require"
	"idena-go/crypto"
	"idena-go/secstore"
	"testing"
)

func Test_GeneratePairs(t *testing.T) {
	secStore := &secstore.SecStore{}
	vc := &ValidationCeremony{
		secStore: secStore,
	}
	key, _ := crypto.GenerateKey()
	secStore.AddKey(crypto.FromECDSA(key))

	for _, tc := range []struct {
		dictionarySize  int
		pairCount       int
		checkUniqueness bool
	}{
		{10, 2, true},
		{3300, 9, true},
		{100, 50, true},
		{10, 20, true},
		{3, 3, true},
		{1, 1, false},
		{3, 4, false},
	} {
		nums, proof := vc.GeneratePairs([]byte("data"), tc.dictionarySize, tc.pairCount)

		require.Equal(t, tc.pairCount*2, len(nums))
		require.NotNil(t, proof)

		for i := 0; i < len(nums); i++ {
			require.True(t, nums[i] < tc.dictionarySize)
		}

		if !tc.checkUniqueness {
			continue
		}

		// Check there is no pair with same values
		for i := 0; i < tc.pairCount; i++ {
			require.NotEqual(t, nums[i*2], nums[i*2+1])
		}

		// Check there is no same pairs
		pairs := mapset.NewSet()
		for i := 0; i < tc.pairCount; i++ {
			require.False(t, pairs.Contains(fmt.Sprintf("%d;%d", nums[i*2], nums[i*2+1])))
			pairs.Add(fmt.Sprintf("%d;%d", nums[i*2], nums[i*2+1]))
			pairs.Add(fmt.Sprintf("%d;%d", nums[i*2+1], nums[i*2]))
		}
	}
}

func Test_CheckPair(t *testing.T) {
	secStore := &secstore.SecStore{}
	vc := &ValidationCeremony{
		secStore: secStore,
	}

	key, _ := crypto.GenerateKey()
	secStore.AddKey(crypto.FromECDSA(key))
	pk := secStore.GetPubKey()
	wrongKey, _ := crypto.GenerateKey()
	seed := []byte("data1")
	dictionarySize := 3300
	pairCount := 9
	nums, proof := vc.GeneratePairs(seed, dictionarySize, pairCount)

	require.True(t, CheckPair(seed, proof, pk, dictionarySize, pairCount, nums[0], nums[1]))
	require.True(t, CheckPair(seed, proof, pk, dictionarySize, pairCount, nums[2], nums[3]))

	require.False(t, CheckPair([]byte("data2"), proof, pk, dictionarySize, 9, nums[0], nums[1]))
	require.False(t, CheckPair(seed, proof, crypto.FromECDSAPub(&wrongKey.PublicKey), dictionarySize, pairCount, nums[0], nums[1]))
	require.False(t, CheckPair(seed, proof, pk, dictionarySize, pairCount, dictionarySize+100, nums[3]))
}

func Test_maxUniquePairs(t *testing.T) {
	require.Equal(t, 0, maxUniquePairs(0))
	require.Equal(t, 0, maxUniquePairs(1))
	require.Equal(t, 1, maxUniquePairs(2))
	require.Equal(t, 3, maxUniquePairs(3))
	require.Equal(t, 6, maxUniquePairs(4))
	require.Equal(t, 10, maxUniquePairs(5))
	require.Equal(t, 15, maxUniquePairs(6))
	require.Equal(t, 21, maxUniquePairs(7))
	require.Equal(t, 28, maxUniquePairs(8))
}
