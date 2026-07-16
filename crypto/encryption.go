package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/idena-network/idena-go/crypto/sha3"
)

// ErrInvalidCiphertext is returned for truncated or unauthenticated encrypted data.
var ErrInvalidCiphertext = errors.New("invalid encrypted payload")

func Encrypt(data []byte, passphrase string) ([]byte, error) {
	key := createHash(passphrase)
	defer zeroBytes(key[:])
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func Decrypt(data []byte, passphrase string) ([]byte, error) {
	key := createHash(passphrase)
	defer zeroBytes(key[:])
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize+gcm.Overhead() {
		return nil, ErrInvalidCiphertext
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}
	return plaintext, nil
}

func createHash(key string) [32]byte {
	return sha3.Sum256([]byte(key))
}
