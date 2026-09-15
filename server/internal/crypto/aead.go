package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

func KEK(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func Seal(kek []byte, plain string) ([]byte, error) {
	if plain == "" {
		return nil, nil
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plain), nil), nil
}

func Open(kek []byte, blob []byte) (string, error) {
	if len(blob) == 0 {
		return "", nil
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return "", fmt.Errorf("cipher too short")
	}
	plain, err := gcm.Open(nil, blob[:ns], blob[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
