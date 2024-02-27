package router

import (
	"p3Chain/common"
	"p3Chain/crypto"
	"p3Chain/crypto/keys"
	"encoding/hex"
	"fmt"
)

// SignContent 使用指定的私钥对内容进行签名
func SignContent(keyDirPath string, keyAddr common.Address, content string) (string, error) {
	// 获取私钥
	keyStore := keys.NewKeyStorePassphrase(keyDirPath)
	key, err := keyStore.GetKey(keyAddr, "")
	if err != nil {
		return "", fmt.Errorf("get key err: %v", err)
	}
	prvKey := key.PrivateKey

	// 生成签名
	h := crypto.Sha3Hash([]byte(content))
	fmt.Println(h)
	sig, err := crypto.SignHash(h, prvKey)
	if err != nil {
		return "", fmt.Errorf("sign err: %v", err)
	}
	// 返回签名的十六进制字符串
	return hex.EncodeToString(sig), nil
}
