package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
)

func TestEmbeddedMainnetResourcesMatchLegacyNode(t *testing.T) {
	header, err := IntermediateGenesisHeader()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sha256Hex(header), "27e696414b955714ba7ed4defe063794c8dcadef28a7e61dd9249b8623571b3c"; got != want {
		t.Fatalf("intermediate genesis header sha256 = %s, want %s", got, want)
	}

	state, err := StateDb()
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	if got, want := sha256Reader(t, state), "7cf6f8c334d76a3617cbd5ac3aa5a104a8d337cb6ceb8d6906c62bf7fab8d131"; got != want {
		t.Fatalf("state snapshot sha256 = %s, want %s", got, want)
	}

	identityState, err := IdentityStateDb()
	if err != nil {
		t.Fatal(err)
	}
	defer identityState.Close()
	if got, want := sha256Reader(t, identityState), "f136ec8939e3f78587a38de517128c7071501e283bac7d12c24ce4be830ff8aa"; got != want {
		t.Fatalf("identity snapshot sha256 = %s, want %s", got, want)
	}
}

func sha256Reader(t *testing.T, reader io.Reader) string {
	t.Helper()
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
