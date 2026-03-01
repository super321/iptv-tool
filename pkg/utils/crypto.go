package utils

import (
	"bytes"
	"crypto/des"
	"encoding/hex"
	"errors"
	"strings"
)

// TripleDESCrypto represents a 3DES encryption/decryption utility using ECB mode.
type TripleDESCrypto struct {
	key []byte
}

// NewTripleDESCrypto creates a new 3DES crypto instance.
func NewTripleDESCrypto(key string) *TripleDESCrypto {
	// Pad or truncate key to exactly 24 bytes
	if len(key) < 24 {
		key += strings.Repeat("0", 24-len(key))
	} else if len(key) > 24 {
		key = key[:24]
	}

	return &TripleDESCrypto{
		key: []byte(key),
	}
}

// pkcs7Padding adds PKCS7 padding to the text
func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// pkcs7UnPadding removes PKCS7 padding from the text
func pkcs7UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, errors.New("invalid padding size")
	}
	unpadding := int(origData[length-1])
	if unpadding > length || unpadding == 0 {
		return nil, errors.New("invalid padding size")
	}
	return origData[:(length - unpadding)], nil
}

// ECBEncrypt encrypts plaintext using 3DES ECB mode and returns hex string.
func (c *TripleDESCrypto) ECBEncrypt(plainText string) (string, error) {
	block, err := des.NewTripleDESCipher(c.key)
	if err != nil {
		return "", err
	}

	origData := []byte(plainText)
	origData = pkcs7Padding(origData, block.BlockSize())
	encrypted := make([]byte, len(origData))

	// ECB mode processes each block independently
	for bs, be := 0, block.BlockSize(); bs < len(origData); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Encrypt(encrypted[bs:be], origData[bs:be])
	}

	return hex.EncodeToString(encrypted), nil
}

// ECBDecrypt decrypts hex ciphertext using 3DES ECB mode and returns plaintext.
func (c *TripleDESCrypto) ECBDecrypt(cipherText string) (string, error) {
	data, err := hex.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := des.NewTripleDESCipher(c.key)
	if err != nil {
		return "", err
	}

	if len(data)%block.BlockSize() != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}

	decrypted := make([]byte, len(data))

	// ECB mode processes each block independently
	for bs, be := 0, block.BlockSize(); bs < len(data); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(decrypted[bs:be], data[bs:be])
	}

	decrypted, err = pkcs7UnPadding(decrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}
