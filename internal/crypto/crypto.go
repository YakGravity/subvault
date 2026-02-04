package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

var (
	magicBytes = []byte("STBK")
	version    = byte(0x01)
	kdfType    = byte(0x01) // Argon2id
)

func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
}

func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key := DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Build file: magic(4) + version(1) + kdf(1) + salt(16) + nonce(12) + ciphertext
	result := make([]byte, 0, 4+1+1+16+12+len(ciphertext))
	result = append(result, magicBytes...)
	result = append(result, version)
	result = append(result, kdfType)
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func Decrypt(data []byte, password string) ([]byte, error) {
	// Minimum size: magic(4) + version(1) + kdf(1) + salt(16) + nonce(12) + tag(16)
	if len(data) < 50 {
		return nil, errors.New("data too short")
	}

	// Verify magic bytes
	if string(data[:4]) != "STBK" {
		return nil, errors.New("invalid file format")
	}

	if data[4] != 0x01 {
		return nil, fmt.Errorf("unsupported version: %d", data[4])
	}

	if data[5] != 0x01 {
		return nil, fmt.Errorf("unsupported KDF: %d", data[5])
	}

	salt := data[6:22]
	nonce := data[22:34]
	ciphertext := data[34:]

	key := DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong password or corrupted data")
	}

	return plaintext, nil
}
