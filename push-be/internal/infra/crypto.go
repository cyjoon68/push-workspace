package infra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
)

type KeyCipher struct {
	key []byte
}

func NewKeyCipher(master string) (*KeyCipher, error) {
	raw, err := base64.StdEncoding.DecodeString(master)
	if err != nil || len(raw) != 32 {
		sum := sha256.Sum256([]byte(master))
		raw = sum[:]
		if master == "" {
			return nil, errors.New("empty master key")
		}
	}
	return &KeyCipher{key: raw}, nil
}

func (k *KeyCipher) Encrypt(plaintext string, userID uuid.UUID, provider string) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	aad := []byte(userID.String() + "|" + provider + "|v1")
	return gcm.Seal(nil, nonce, []byte(plaintext), aad), nonce, nil
}

func (k *KeyCipher) Decrypt(ciphertext, nonce []byte, userID uuid.UUID, provider string) (string, error) {
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	aad := []byte(userID.String() + "|" + provider + "|v1")
	pt, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
