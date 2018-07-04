package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

var key string = "hello world!"

// Two equivalent ways to compute md5 checksum (16 bytes)
func md5HashV1(key string) string {
	h := md5.New()
	h.Write([]byte(key))
	v := h.Sum(nil)
	return hex.EncodeToString(v)
}

func md5HashV2(key string) string {
	v := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", v)
}

func TestMD5Hash(t *testing.T) {
	v1Digest := md5HashV1(key)
	v2Digest := md5HashV2(key)

	log.Printf("MD5: v1Digest = %s, v2Digest = %s\n", v1Digest, v2Digest)

	assert.Equal(t, len(v1Digest), 32)
	assert.Equal(t, len(v2Digest), 32)

	assert.Equal(t, v1Digest, v2Digest)
}

// Two equivalent ways to compute sha1 checksum (16 bytes)
func sha1HashV1(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	v := h.Sum(nil)
	return hex.EncodeToString(v)
}

func sha1HashV2(key string) string {
	v := sha1.Sum([]byte(key))
	return fmt.Sprintf("%x", v)
}

func TestSHA1Hash(t *testing.T) {
	v1Digest := sha1HashV1(key)
	v2Digest := sha1HashV2(key)

	log.Printf("SHA1: v1Digest = %s, v2Digest = %s\n", v1Digest, v2Digest)

	assert.Equal(t, len(v1Digest), 40)
	assert.Equal(t, len(v2Digest), 40)

	assert.Equal(t, v1Digest, v2Digest)
}

// Two equivalent ways to compute sha256 checksum (32 bytes)
func sha256HashV1(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	v := h.Sum(nil)
	return hex.EncodeToString(v)
}

func sha256HashV2(key string) string {
	v := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", v)
}

func TestSHA256Hash(t *testing.T) {
	v1Digest := sha256HashV1(key)
	v2Digest := sha256HashV2(key)

	log.Printf("SHA256: v1Digest = %s, v2Digest = %s\n", v1Digest, v2Digest)

	assert.Equal(t, len(v1Digest), 64)
	assert.Equal(t, len(v2Digest), 64)

	assert.Equal(t, v1Digest, v2Digest)
}
