package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/idena-network/idena-go/crypto"
)

type dnaSignDoubleHashVector struct {
	Format         string `json:"format"`
	Value          string `json:"value"`
	FirstHashHex   string `json:"first_hash_hex"`
	SigningHashHex string `json:"signing_hash_hex"`
	SignatureHex   string `json:"signature_hex"`
	Address        string `json:"address"`
}

func TestDnaSignDoubleHashCompatibilityVector(t *testing.T) {
	data, err := os.ReadFile("testdata/dna_sign_double_hash.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector dnaSignDoubleHashVector
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&vector); err != nil {
		t.Fatal(err)
	}

	if vector.Format != DoubleHash {
		t.Fatalf("unexpected signing format: got %q, want %q", vector.Format, DoubleHash)
	}
	if got := signedDataFormatOrDefault(nil); got != DoubleHash {
		t.Fatalf("unexpected default signing format: got %q, want %q", got, DoubleHash)
	}

	firstHash := crypto.Hash([]byte(vector.Value))
	if got := hex.EncodeToString(firstHash[:]); got != vector.FirstHashHex {
		t.Fatalf("first Keccak-256 mismatch: got %s, want %s", got, vector.FirstHashHex)
	}
	digest, err := signatureHash(vector.Value, vector.Format)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(digest[:]); got != vector.SigningHashHex {
		t.Fatalf("double Keccak-256 mismatch: got %s, want %s", got, vector.SigningHashHex)
	}

	signature, err := hex.DecodeString(vector.SignatureHex)
	if err != nil {
		t.Fatal(err)
	}
	if len(signature) != 65 {
		t.Fatalf("unexpected signature length: got %d, want 65", len(signature))
	}
	pubKey, err := crypto.Ecrecover(digest[:], signature)
	if err != nil {
		t.Fatal(err)
	}
	address, err := crypto.PubKeyBytesToAddress(pubKey)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(address.Hex(), vector.Address) {
		t.Fatalf("recovered address mismatch: got %s, want %s", address.Hex(), vector.Address)
	}

	prefixDigest, err := signatureHash(vector.Value, Prefix)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(prefixDigest[:], digest[:]) {
		t.Fatal("prefix and doubleHash signing formats produced the same digest")
	}
}
