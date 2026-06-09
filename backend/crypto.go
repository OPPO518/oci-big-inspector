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
	"path/filepath"
)

// 自动生成或读取本地持久化的专属加密密钥
func getEncryptionKey() []byte {
	keyPath := "/app/data/secret.key"
	// 本地调试兼容
	if _, err := os.Stat("/app/data"); os.IsNotExist(err) {
		keyPath = "./data/secret.key"
	}

	// 如果密钥文件不存在，说明是首次启动，自动生成一个 32 字节的强随机密钥
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		token := make([]byte, 32)
		_, _ = rand.Read(token)
		hexToken := hex.EncodeToString(token)
		_ = os.WriteFile(keyPath, []byte(hexToken), 0600)
	}

	// 读取本地固化的密钥
	content, err := os.ReadFile(keyPath)
	if err != nil {
		// 极度罕见保底
		content = []byte("OCI_BIG_INSPECTOR_DEFAULT_FALLBACK")
	}

	hash := sha256.Sum256(content)
	return hash[:]
}

// EncryptText 保持不变
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

// DecryptText 保持不变
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
		return errors.New("解密失败")
	}
	return string(plainText), nil
}
