package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

var AllAnimeKey []byte

func init() {
	hash := sha256.Sum256([]byte("Xot36i3lK3:v1"))
	AllAnimeKey = hash[:]
}

// DecryptAllAnime decrypts the tobeparsed string from AllAnime API response
func DecryptAllAnime(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	if len(data) < 13+16 {
		return "", errors.New("data too short")
	}

	// IV (12 bytes from offset 1 to 12)
	ivBytes := data[1:13]
	
	// OpenSSL AES-256-CTR IV is 16 bytes: 12 bytes iv + 4 bytes counter (0x00000002)
	iv := make([]byte, 16)
	copy(iv[0:12], ivBytes)
	iv[12] = 0x00
	iv[13] = 0x00
	iv[14] = 0x00
	iv[15] = 0x02

	ciphertext := data[13 : len(data)-16]

	block, err := aes.NewCipher(AllAnimeKey)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}

// DecryptFilemoon decrypts Filemoon payload
func DecryptFilemoon(payload, ivB64, keyB64Parts []string) (string, error) {
	// Not immediately needed, but we will write it if we encounter Filemoon
	return "", errors.New("not implemented")
}

// HexToBytes helper
func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
