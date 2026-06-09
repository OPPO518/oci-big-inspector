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

// 获取经 SHA256 固定为 32 字节的 AES-256 密钥，防止用户填入的 MASTER_KEY 长度不合规
func getEncryptionKey() []byte {
	masterKey := os.Getenv("MASTER_KEY")
	if masterKey == "" {
		// 如果用户没传，保底用一个，但在生产环境下单容器启动时必须通过 -e MASTER_KEY 传入
		masterKey = "OCI_BIG_INSPECTOR_DEFAULT_FALLBACK_KEY_CHANGE_ME"
	}
	hash := sha256.Sum256([]byte(masterKey))
	return hash[:]
}

// EncryptText 使用 AES-GCM 加密明文文本，返回十六进制密文字符串
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

	// 创建随机盐值/Nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(cipherText), nil
}

// DecryptText 解密十六进制密文字符串，返回原始明文
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
		return "", errors.New("解密失败：可能 MASTER_KEY 不正确或数据被篡改")
	}

	return string(plainText), nil
}
