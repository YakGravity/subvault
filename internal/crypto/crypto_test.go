package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	plaintext := []byte(`{"subscriptions": [{"name": "Netflix", "cost": 15.99}]}`)
	password := "test-password-123"

	encrypted, err := Encrypt(plaintext, password)
	require.NoError(t, err)
	assert.True(t, len(encrypted) > len(plaintext))

	// Verify header
	assert.Equal(t, "STBK", string(encrypted[:4]))
	assert.Equal(t, byte(0x01), encrypted[4])
	assert.Equal(t, byte(0x01), encrypted[5])

	decrypted, err := Decrypt(encrypted, password)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecryptWrongPassword(t *testing.T) {
	plaintext := []byte("secret data")
	encrypted, err := Encrypt(plaintext, "correct-password")
	require.NoError(t, err)

	_, err = Decrypt(encrypted, "wrong-password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decryption failed")
}

func TestDecryptCorruptedData(t *testing.T) {
	plaintext := []byte("secret data")
	encrypted, err := Encrypt(plaintext, "password")
	require.NoError(t, err)

	// Corrupt the ciphertext
	encrypted[len(encrypted)-1] ^= 0xFF

	_, err = Decrypt(encrypted, "password")
	assert.Error(t, err)
}

func TestDecryptTooShort(t *testing.T) {
	_, err := Decrypt([]byte("short"), "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDecryptInvalidMagic(t *testing.T) {
	data := make([]byte, 50)
	copy(data[:4], "XXXX")

	_, err := Decrypt(data, "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file format")
}

func TestEncryptEmpty(t *testing.T) {
	encrypted, err := Encrypt([]byte{}, "password")
	require.NoError(t, err)

	decrypted, err := Decrypt(encrypted, "password")
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}
