package auth

import (
	"encoding/hex"
	"testing"
)

func TestInitJWTSecret_CustomSecret(t *testing.T) {
	InitJWTSecret("my-custom-secret-for-test")
	if string(jwtSecret) != "my-custom-secret-for-test" {
		t.Errorf("expected custom secret, got %q", string(jwtSecret))
	}
}

func TestInitJWTSecret_AutoGenerate(t *testing.T) {
	InitJWTSecret("")
	if len(jwtSecret) != 32 {
		t.Errorf("auto-generated secret should be 32 bytes, got %d", len(jwtSecret))
	}
}

func TestGetJWTSecretHex(t *testing.T) {
	InitJWTSecret("test-secret-0123456789ab")
	hexStr := GetJWTSecretHex()

	// Should be valid hex
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("GetJWTSecretHex returned invalid hex: %v", err)
	}
	if string(decoded) != "test-secret-0123456789ab" {
		t.Errorf("hex decode mismatch: got %q", string(decoded))
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	InitJWTSecret("test-jwt-secret-for-unit-test!")

	token, err := GenerateToken(42, "admin")
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("Username = %q, want %q", claims.Username, "admin")
	}
	if claims.Issuer != "iptv-tool-v2" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "iptv-tool-v2")
	}
}

func TestParseToken_Invalid(t *testing.T) {
	InitJWTSecret("test-jwt-secret-for-unit-test!")

	_, err := ParseToken("totally.invalid.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	// Generate token with one secret
	InitJWTSecret("secret-one-for-generation!")
	token, err := GenerateToken(1, "user")
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	// Switch to different secret and try to parse
	InitJWTSecret("secret-two-for-validation!")
	_, err = ParseToken(token)
	if err == nil {
		t.Error("expected error when parsing with wrong secret")
	}
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestGenerateToken_DifferentUsers(t *testing.T) {
	InitJWTSecret("test-jwt-secret-for-unique!")

	token1, _ := GenerateToken(1, "user1")
	token2, _ := GenerateToken(2, "user2")

	if token1 == token2 {
		t.Error("different users should produce different tokens")
	}
}
