package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"sync"
)

var (
	rsaPrivateKey *rsa.PrivateKey
	rsaPublicKey  []byte // PEM encoded public key
	rsaOnce       sync.Once
)

// initRSA generates a new RSA key pair for the application's lifecycle
func initRSA() {
	var err error
	rsaPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		slog.Error("Failed to generate RSA key pair", "error", err)
		panic("Cannot generate RSA key pair for security")
	}

	pubASN1, err := x509.MarshalPKIXPublicKey(&rsaPrivateKey.PublicKey)
	if err != nil {
		slog.Error("Failed to marshal RSA public key", "error", err)
		panic("Cannot marshal RSA public key")
	}

	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	rsaPublicKey = pem.EncodeToMemory(pubBlock)
	slog.Info("RSA key pair initialized for secure login")
}

// GetRSAPublicKey returns the PEM encoded public key
func GetRSAPublicKey() string {
	rsaOnce.Do(initRSA)
	return string(rsaPublicKey)
}

// DecryptRSA decrypts a base64 encoded RSA-OAEP encrypted string
func DecryptRSA(cipherTextBase64 string) (string, error) {
	rsaOnce.Do(initRSA)

	cipherText, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encoding")
	}

	// Web Crypto API typically uses SHA-256 for OAEP padding
	hash := sha256.New()

	plainText, err := rsa.DecryptOAEP(hash, rand.Reader, rsaPrivateKey, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return string(plainText), nil
}
