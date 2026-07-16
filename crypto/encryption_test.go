package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"strconv"
	"testing"

	"github.com/idena-network/idena-go/crypto/sha3"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	data := []byte{0x1, 0x2, 0x3}
	pass := "123456abc"
	encrypted, err := Encrypt(data, pass)
	require.NoError(t, err)
	require.NotEqual(t, data, encrypted)
	decrypted, err := Decrypt(encrypted, pass)
	require.NoError(t, err)

	require.Equal(t, data, decrypted)
}

func TestDecryptSupportsLegacyCiphertext(t *testing.T) {
	data := []byte("legacy exported node key")
	passphrase := "correct horse battery staple"
	key := sha3.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	ciphertext := gcm.Seal(append([]byte{}, nonce...), nonce, data, nil)

	decrypted, err := Decrypt(ciphertext, passphrase)

	require.NoError(t, err)
	require.Equal(t, data, decrypted)
}

func TestDecryptRejectsInvalidCiphertext(t *testing.T) {
	for size := 0; size < 28; size++ {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			_, err := Decrypt(bytes.Repeat([]byte{0x42}, size), "password")
			require.ErrorIs(t, err, ErrInvalidCiphertext)
		})
	}

	encrypted, err := Encrypt([]byte("secret"), "correct password")
	require.NoError(t, err)
	_, err = Decrypt(encrypted, "wrong password")
	require.ErrorIs(t, err, ErrInvalidCiphertext)

	encrypted[len(encrypted)-1] ^= 0xff
	_, err = Decrypt(encrypted, "correct password")
	require.ErrorIs(t, err, ErrInvalidCiphertext)
}

func FuzzDecryptDoesNotPanic(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{0x42}, 28))
	f.Fuzz(func(t *testing.T, ciphertext []byte) {
		_, err := Decrypt(ciphertext, "password")
		if err != nil && !errors.Is(err, ErrInvalidCiphertext) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
