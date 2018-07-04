package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log"
	"testing"

	"golang.org/x/crypto/ed25519"

	"github.com/stretchr/testify/assert"
)

var plainText string = "here are some plain text"

// AES symmetric (private) encryption
func TestAESEncryption(t *testing.T) {
	digest := md5HashV1(key)

	// encryption
	block, err := aes.NewCipher([]byte(digest)) // AES key, either 16, 24, or 32 bytes
	assert.Equal(t, err, nil)

	gcm, err := cipher.NewGCM(block)
	assert.Equal(t, err, nil)

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	assert.Equal(t, err, nil)

	// append the encrypted text behind nonce
	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	log.Printf("AES: cipherText = %x\n", cipherText)

	// decryption
	block, err = aes.NewCipher([]byte(digest)) // AES key, either 16, 24, or 32 bytes
	assert.Equal(t, err, nil)

	gcm, err = cipher.NewGCM(block)
	assert.Equal(t, err, nil)

	nonceSize := gcm.NonceSize()
	nonce, secretText := cipherText[:nonceSize], cipherText[nonceSize:]
	originalText, err := gcm.Open(nil, nonce, secretText, nil)

	assert.Equal(t, err, nil)
	assert.Equal(t, originalText, []byte(plainText))
}

// RSA encryption
func TestRSAEncryption(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	assert.Equal(t, nil, err)

	publicKey := privateKey.PublicKey

	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, &publicKey, []byte(plainText))
	assert.Equal(t, nil, err)

	log.Printf("RSA: cipherText = %x\n", cipherText)

	originalText, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, cipherText)
	assert.Equal(t, nil, err)
	assert.Equal(t, plainText, string(originalText))
}

// key pair generation
func TestKeyPair(t *testing.T) {
	var seed [32]byte

	_, err := io.ReadFull(rand.Reader, seed[:])
	assert.Equal(t, nil, err)

	privateKey := ed25519.NewKeyFromSeed(seed[:])
	publicKey := privateKey.Public()

	log.Printf("ED25519: Seed = %x, Address = %x\n", seed, publicKey)

	message := []byte("test message")

	signature := ed25519.Sign(privateKey, message)

	pubKey := publicKey.(ed25519.PublicKey)
	verify := ed25519.Verify(pubKey, message, signature)
	assert.Equal(t, true, verify)
}
