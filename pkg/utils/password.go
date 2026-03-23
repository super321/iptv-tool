package utils

import (
	"crypto/rand"
	"math/big"
)

const (
	lowerChars   = "abcdefghijklmnopqrstuvwxyz"
	upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars   = "0123456789"
	specialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	allChars     = lowerChars + upperChars + digitChars + specialChars
)

// GenerateRandomPassword generates a cryptographically random password of the
// specified length. The password is guaranteed to contain at least one lowercase
// letter, one uppercase letter, one digit, and one special character.
// Panics if length < 4 (cannot satisfy all character-class requirements).
func GenerateRandomPassword(length int) string {
	if length < 4 {
		panic("password length must be at least 4 to satisfy all character-class requirements")
	}

	password := make([]byte, length)

	// Guarantee one character from each required class in random positions.
	mandatory := []string{lowerChars, upperChars, digitChars, specialChars}
	usedPositions := make(map[int]bool, len(mandatory))
	for _, charset := range mandatory {
		pos := cryptoRandIntn(length)
		for usedPositions[pos] {
			pos = cryptoRandIntn(length)
		}
		usedPositions[pos] = true
		password[pos] = charset[cryptoRandIntn(len(charset))]
	}

	// Fill remaining positions with random characters from the full set.
	for i := 0; i < length; i++ {
		if !usedPositions[i] {
			password[i] = allChars[cryptoRandIntn(len(allChars))]
		}
	}

	return string(password)
}

// cryptoRandIntn returns a cryptographically random int in [0, n).
func cryptoRandIntn(n int) int {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return int(val.Int64())
}
