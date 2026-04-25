package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
)

const (
	presetKey = "0CoJUm6Qyw8W8jud"
	iv        = "0102030405060708"
	pubKey    = "010001"
	modulus   = "00e0b509f6259df8642dbc35662901477df22677ec152b5ff68ace615bb7b725152b3ab17a876aea8a5aa76d2e417629ec4ee341f56135fccf695280104e0312ecbda92557c93870114af6c9d05c4f7f0c3685b7a46bee255932575cce10b424d813cfe4875d3e82047b97ddef52741d546b8e289dc6935b3ece0462db0a22b8e7"
	keyChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type WeAPIForm struct {
	Params    string
	EncSecKey string
}

func EncryptWeAPI(plain []byte) (WeAPIForm, error) {
	secret, err := randomSecret(16)
	if err != nil {
		return WeAPIForm{}, err
	}
	return EncryptWeAPIWithSecret(plain, secret)
}

func EncryptWeAPIWithSecret(plain []byte, secret string) (WeAPIForm, error) {
	if len(secret) != 16 {
		return WeAPIForm{}, fmt.Errorf("secret 长度必须为 16，当前为 %d", len(secret))
	}
	first, err := aesCBCEncrypt(plain, []byte(presetKey), []byte(iv))
	if err != nil {
		return WeAPIForm{}, err
	}
	second, err := aesCBCEncrypt([]byte(first), []byte(secret), []byte(iv))
	if err != nil {
		return WeAPIForm{}, err
	}
	encSecKey, err := rsaEncrypt(secret)
	if err != nil {
		return WeAPIForm{}, err
	}
	return WeAPIForm{Params: second, EncSecKey: encSecKey}, nil
}

func aesCBCEncrypt(plain, key, ivBytes []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(ivBytes) != block.BlockSize() {
		return "", errors.New("AES IV 长度无效")
	}
	padded := pkcs7Pad(plain, block.BlockSize())
	cipherText := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, ivBytes)
	mode.CryptBlocks(cipherText, padded)
	return base64Encode(cipherText), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	if blockSize <= 0 {
		return data
	}
	padding := blockSize - len(data)%blockSize
	return append(append([]byte{}, data...), bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func rsaEncrypt(secret string) (string, error) {
	reversed := reverseString(secret)
	m, ok := new(big.Int).SetString(modulus, 16)
	if !ok {
		return "", errors.New("RSA modulus 无效")
	}
	e, ok := new(big.Int).SetString(pubKey, 16)
	if !ok {
		return "", errors.New("RSA pubKey 无效")
	}
	text := new(big.Int).SetBytes([]byte(reversed))
	encrypted := new(big.Int).Exp(text, e, m)
	out := encrypted.Text(16)
	if len(out) > 256 {
		return "", fmt.Errorf("RSA 输出长度异常: %d", len(out))
	}
	for len(out) < 256 {
		out = "0" + out
	}
	return out, nil
}

func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func randomSecret(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	max := big.NewInt(int64(len(keyChars)))
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = keyChars[idx.Int64()]
	}
	return string(out), nil
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
