package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"sync"
	"testing"
)

// resetRSAState resets the package-level RSA state so tests are independent.
func resetRSAState() {
	rsaPrivateKey = nil
	rsaPublicKey = nil
	rsaOnce = sync.Once{}
}

func TestGetRSAPublicKey_ValidPEM(t *testing.T) {
	resetRSAState()

	pubKeyPEM := GetRSAPublicKey()

	if pubKeyPEM == "" {
		t.Fatal("GetRSAPublicKey returned empty string")
	}

	if !strings.Contains(pubKeyPEM, "-----BEGIN PUBLIC KEY-----") {
		t.Error("public key PEM missing BEGIN marker")
	}
	if !strings.Contains(pubKeyPEM, "-----END PUBLIC KEY-----") {
		t.Error("public key PEM missing END marker")
	}

	// Parse to verify it's valid
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse public key: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatal("parsed key is not RSA")
	}

	if rsaPub.N.BitLen() < 2048 {
		t.Errorf("RSA key size = %d bits, want >= 2048", rsaPub.N.BitLen())
	}
}

func TestGetRSAPublicKey_Idempotent(t *testing.T) {
	resetRSAState()

	key1 := GetRSAPublicKey()
	key2 := GetRSAPublicKey()

	if key1 != key2 {
		t.Error("GetRSAPublicKey should return the same key on subsequent calls")
	}
}

func TestDecryptRSA_OAEP(t *testing.T) {
	resetRSAState()

	// Initialize key pair
	_ = GetRSAPublicKey()

	// Encrypt with the public key using OAEP (same as Web Crypto API)
	plaintext := "my-secret-password-123"
	hash := sha256.New()
	cipherText, err := rsa.EncryptOAEP(hash, rand.Reader, &rsaPrivateKey.PublicKey, []byte(plaintext), nil)
	if err != nil {
		t.Fatalf("rsa.EncryptOAEP error: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(cipherText)

	decrypted, err := DecryptRSA(encoded)
	if err != nil {
		t.Fatalf("DecryptRSA error: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptRSA = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptRSA_PKCS1v15_Fallback(t *testing.T) {
	resetRSAState()

	// Initialize key pair
	_ = GetRSAPublicKey()

	// Encrypt with PKCS1v15 (JSEncrypt fallback path)
	plaintext := "fallback-password"
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, &rsaPrivateKey.PublicKey, []byte(plaintext))
	if err != nil {
		t.Fatalf("rsa.EncryptPKCS1v15 error: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(cipherText)

	decrypted, err := DecryptRSA(encoded)
	if err != nil {
		t.Fatalf("DecryptRSA error: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptRSA = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptRSA_InvalidBase64(t *testing.T) {
	resetRSAState()
	_ = GetRSAPublicKey()

	_, err := DecryptRSA("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64 input")
	}
}

func TestDecryptRSA_InvalidCiphertext(t *testing.T) {
	resetRSAState()
	_ = GetRSAPublicKey()

	// Valid base64 but not valid RSA ciphertext
	encoded := base64.StdEncoding.EncodeToString([]byte("this is not encrypted data at all"))
	_, err := DecryptRSA(encoded)
	if err == nil {
		t.Error("expected error for invalid ciphertext")
	}
}
