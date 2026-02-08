package crypto_test

import (
	"testing"

	"github.com/mathias/boeuf/internal/crypto"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	// ARRANGE
	key := "12345678901234567890123456789012" // 32 bytes
	plaintext := "my_sensitive_refresh_token"

	// ACT - Encrypt
	ciphertext, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// ASSERT - Ciphertext should not equal plaintext
	if ciphertext == plaintext {
		t.Error("Ciphertext should not equal plaintext")
	}

	// ACT - Decrypt
	decrypted, err := crypto.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// ASSERT - Decrypted should equal original plaintext
	if decrypted != plaintext {
		t.Errorf("Expected %s, got %s", plaintext, decrypted)
	}
}

func TestEncryptWithInvalidKey(t *testing.T) {
	// ARRANGE
	key := "tooshort" // Not 32 bytes
	plaintext := "test"

	// ACT
	_, err := crypto.Encrypt(plaintext, key)

	// ASSERT
	if err == nil {
		t.Error("Expected error with invalid key length")
	}
}

func TestDecryptWithInvalidKey(t *testing.T) {
	// ARRANGE
	key := "12345678901234567890123456789012"
	plaintext := "test"
	ciphertext, _ := crypto.Encrypt(plaintext, key)

	wrongKey := "00000000000000000000000000000000"

	// ACT
	_, err := crypto.Decrypt(ciphertext, wrongKey)

	// ASSERT
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

func TestEncryptDifferentOutputs(t *testing.T) {
	// ARRANGE
	key := "12345678901234567890123456789012"
	plaintext := "test"

	// ACT
	ciphertext1, _ := crypto.Encrypt(plaintext, key)
	ciphertext2, _ := crypto.Encrypt(plaintext, key)

	// ASSERT - Should produce different ciphertexts due to random nonce
	if ciphertext1 == ciphertext2 {
		t.Error("Expected different ciphertexts due to random nonce (GCM)")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	// ARRANGE
	key := "12345678901234567890123456789012"
	invalidCiphertext := "not_valid_base64_or_ciphertext"

	// ACT
	_, err := crypto.Decrypt(invalidCiphertext, key)

	// ASSERT
	if err == nil {
		t.Error("Expected error when decrypting invalid ciphertext")
	}
}
