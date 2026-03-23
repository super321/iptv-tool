package utils

import (
	"strings"
	"testing"
	"unicode"
)

func TestGenerateRandomPassword_Length(t *testing.T) {
	for _, length := range []int{4, 8, 12, 16, 32} {
		pw := GenerateRandomPassword(length)
		if len(pw) != length {
			t.Errorf("expected length %d, got %d (password: %q)", length, len(pw), pw)
		}
	}
}

func TestGenerateRandomPassword_CharacterClasses(t *testing.T) {
	// Run multiple times to reduce flakiness risk from randomness.
	for i := 0; i < 50; i++ {
		pw := GenerateRandomPassword(8)

		var hasLower, hasUpper, hasDigit, hasSpecial bool
		for _, ch := range pw {
			switch {
			case unicode.IsLower(ch):
				hasLower = true
			case unicode.IsUpper(ch):
				hasUpper = true
			case unicode.IsDigit(ch):
				hasDigit = true
			case strings.ContainsRune(specialChars, ch):
				hasSpecial = true
			}
		}

		if !hasLower {
			t.Errorf("password %q missing lowercase letter", pw)
		}
		if !hasUpper {
			t.Errorf("password %q missing uppercase letter", pw)
		}
		if !hasDigit {
			t.Errorf("password %q missing digit", pw)
		}
		if !hasSpecial {
			t.Errorf("password %q missing special character", pw)
		}
	}
}

func TestGenerateRandomPassword_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pw := GenerateRandomPassword(8)
		if seen[pw] {
			t.Errorf("duplicate password generated: %q", pw)
		}
		seen[pw] = true
	}
}

func TestGenerateRandomPassword_PanicOnShortLength(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for length < 4, but did not panic")
		}
	}()
	GenerateRandomPassword(3)
}
