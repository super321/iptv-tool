package utils

import "testing"

func TestTripleDESCrypto_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		plaintext string
	}{
		{"short plaintext", "12345678901234567890abcd", "hello"},
		{"exact block size", "12345678901234567890abcd", "12345678"},                         // 8 bytes = 1 block
		{"multi block", "12345678901234567890abcd", "hello world, this is a longer message"}, // >1 block
		{"empty plaintext", "12345678901234567890abcd", ""},
		{"chinese text", "12345678901234567890abcd", "你好世界"},
		{"special chars", "12345678901234567890abcd", "a$b$c$d$e$f$g$CTC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto := NewTripleDESCrypto(tt.key)

			encrypted, err := crypto.ECBEncrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("ECBEncrypt(%q) error: %v", tt.plaintext, err)
			}

			if encrypted == "" && tt.plaintext != "" {
				t.Fatalf("ECBEncrypt(%q) returned empty string", tt.plaintext)
			}

			decrypted, err := crypto.ECBDecrypt(encrypted)
			if err != nil {
				t.Fatalf("ECBDecrypt(%q) error: %v", encrypted, err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("round trip failed: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestTripleDESCrypto_KeyPadding(t *testing.T) {
	// Short key should be padded with '0' to 24 bytes
	shortCrypto := NewTripleDESCrypto("abc")
	encrypted, err := shortCrypto.ECBEncrypt("test")
	if err != nil {
		t.Fatalf("ECBEncrypt with short key error: %v", err)
	}
	decrypted, err := shortCrypto.ECBDecrypt(encrypted)
	if err != nil {
		t.Fatalf("ECBDecrypt with short key error: %v", err)
	}
	if decrypted != "test" {
		t.Errorf("got %q, want %q", decrypted, "test")
	}
}

func TestTripleDESCrypto_KeyTruncation(t *testing.T) {
	// Long key should be truncated to 24 bytes
	longCrypto := NewTripleDESCrypto("12345678901234567890abcdefghijk")
	encrypted, err := longCrypto.ECBEncrypt("test")
	if err != nil {
		t.Fatalf("ECBEncrypt with long key error: %v", err)
	}
	decrypted, err := longCrypto.ECBDecrypt(encrypted)
	if err != nil {
		t.Fatalf("ECBDecrypt with long key error: %v", err)
	}
	if decrypted != "test" {
		t.Errorf("got %q, want %q", decrypted, "test")
	}
}

func TestTripleDESCrypto_DifferentKeysProduceDifferentCiphertext(t *testing.T) {
	crypto1 := NewTripleDESCrypto("key1key1key1key1key1key1")
	crypto2 := NewTripleDESCrypto("key2key2key2key2key2key2")

	enc1, err := crypto1.ECBEncrypt("same plaintext")
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := crypto2.ECBEncrypt("same plaintext")
	if err != nil {
		t.Fatal(err)
	}

	if enc1 == enc2 {
		t.Error("different keys should produce different ciphertext")
	}
}

func TestTripleDESCrypto_DecryptInvalidHex(t *testing.T) {
	crypto := NewTripleDESCrypto("12345678901234567890abcd")

	// Invalid hex string
	_, err := crypto.ECBDecrypt("not_hex_at_all!")
	if err == nil {
		t.Error("expected error for invalid hex input")
	}
}

func TestTripleDESCrypto_DecryptWrongBlockSize(t *testing.T) {
	crypto := NewTripleDESCrypto("12345678901234567890abcd")

	// Valid hex but not a multiple of block size (odd number of hex chars → 3 bytes)
	_, err := crypto.ECBDecrypt("abcdef")
	if err == nil {
		t.Error("expected error for input not aligned to block size")
	}
}

func TestTripleDESCrypto_EncryptOutputIsHex(t *testing.T) {
	crypto := NewTripleDESCrypto("12345678901234567890abcd")
	encrypted, err := crypto.ECBEncrypt("test")
	if err != nil {
		t.Fatal(err)
	}

	// Verify the output only contains valid hex characters
	for _, ch := range encrypted {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("encrypted output contains non-hex character: %c", ch)
		}
	}
}
