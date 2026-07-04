package bn256

import (
	"bytes"
	"math/big"
	"testing"

	cloudflare "github.com/idena-network/idena-go/crypto/bn256/cloudflare"
	google "github.com/idena-network/idena-go/crypto/bn256/google"
	"github.com/stretchr/testify/require"
)

func TestPairingMatchesGoogleImplementation(t *testing.T) {
	scalars := []*big.Int{
		big.NewInt(1),
		big.NewInt(2),
		big.NewInt(37),
		big.NewInt(999),
	}

	for _, g1Scalar := range scalars {
		for _, g2Scalar := range scalars {
			cloudflareG1 := new(cloudflare.G1).ScalarBaseMult(g1Scalar)
			cloudflareG2 := new(cloudflare.G2).ScalarBaseMult(g2Scalar)
			googleG1 := new(google.G1).ScalarBaseMult(g1Scalar)
			googleG2 := new(google.G2).ScalarBaseMult(g2Scalar)

			require.True(t,
				bytes.Equal(
					cloudflare.Pair(cloudflareG1, cloudflareG2).Marshal(),
					google.Pair(googleG1, googleG2).Marshal(),
				),
				"pairing mismatch for G1 scalar %s and G2 scalar %s",
				g1Scalar,
				g2Scalar,
			)
		}
	}
}

func TestPairingCheckMatchesGoogleImplementation(t *testing.T) {
	cloudflareG1 := []*cloudflare.G1{
		new(cloudflare.G1).ScalarBaseMult(big.NewInt(1)),
		new(cloudflare.G1).ScalarBaseMult(new(big.Int).Sub(cloudflare.Order, big.NewInt(1))),
	}
	cloudflareG2 := []*cloudflare.G2{
		new(cloudflare.G2).ScalarBaseMult(big.NewInt(1)),
		new(cloudflare.G2).ScalarBaseMult(big.NewInt(1)),
	}
	googleG1 := []*google.G1{
		new(google.G1).ScalarBaseMult(big.NewInt(1)),
		new(google.G1).ScalarBaseMult(new(big.Int).Sub(google.Order, big.NewInt(1))),
	}
	googleG2 := []*google.G2{
		new(google.G2).ScalarBaseMult(big.NewInt(1)),
		new(google.G2).ScalarBaseMult(big.NewInt(1)),
	}

	require.Equal(t,
		google.PairingCheck(googleG1, googleG2),
		cloudflare.PairingCheck(cloudflareG1, cloudflareG2),
	)
}
