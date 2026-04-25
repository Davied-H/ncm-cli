package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"
	"testing"
)

func TestPKCS7Pad(t *testing.T) {
	got := pkcs7Pad([]byte("abc"), 16)
	if len(got) != 16 {
		t.Fatalf("len = %d, want 16", len(got))
	}
	if got[len(got)-1] != 13 {
		t.Fatalf("last padding byte = %d, want 13", got[len(got)-1])
	}
}

func TestAESCBCEncryptCanDecrypt(t *testing.T) {
	plain := []byte(`{"hello":"world"}`)
	encrypted, err := aesCBCEncrypt(plain, []byte(presetKey), []byte(iv))
	if err != nil {
		t.Fatal(err)
	}
	cipherText, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher([]byte(presetKey))
	if err != nil {
		t.Fatal(err)
	}
	decrypted := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(block, []byte(iv)).CryptBlocks(decrypted, cipherText)
	if !bytes.HasPrefix(decrypted, plain) {
		t.Fatalf("decrypted prefix = %q, want %q", decrypted[:len(plain)], plain)
	}
}

func TestEncryptWeAPIWithSecret(t *testing.T) {
	form, err := EncryptWeAPIWithSecret([]byte(`{"csrf_token":"x"}`), "0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if form.Params == "" {
		t.Fatal("Params is empty")
	}
	if len(form.EncSecKey) != 256 {
		t.Fatalf("EncSecKey length = %d, want 256", len(form.EncSecKey))
	}
	if strings.Contains(form.Params, "csrf") {
		t.Fatal("Params should be encrypted")
	}
}
