package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

func getEncryptionKey() []byte {
	keyPath := "/app/data/secret.key"
	if _, err := os.Stat("/app/data"); os.IsNotExist(err) {
		keyPath = "./data/secret.key"
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		token := make([]byte, 32)
		_, _ = rand.Read(token)
		hexToken := hex.EncodeToString(token)
		_ = os.WriteFile(keyPath, []byte(hexToken), 0600)
	}
	content, err := os.ReadFile(keyPath)
	if err != nil {
		content = []byte("OCI_BIG_INSPECTOR_DEFAULT_FALLBACK")
	}
	hash := sha256.Sum256(content)
	return hash[:]
}

func EncryptText(plainText string) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(cipherText), nil
}

func DecryptText(cryptoHex string) (string, error) {
	key := getEncryptionKey()
	data, err := hex.DecodeString(cryptoHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("密文长度异常")
	}
	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		// 修复点：严格返回两个值
		return "", errors.New("解密失败")
	}
	return string(plainText), nil
}
