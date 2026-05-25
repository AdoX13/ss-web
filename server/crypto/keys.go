package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	// MasterKeyEnv is the environment variable used by the API to load the
	// 256-bit key-encryption key for PHI field encryption.
	MasterKeyEnv = "MEDSEC_MASTER_KEY"

	fieldKeySize = 32
)

// LoadMasterKeyFromEnv reads MEDSEC_MASTER_KEY. The value may be raw base64,
// hex, or a 32-byte development string. Production should use base64.
func LoadMasterKeyFromEnv() ([]byte, error) {
	return ParseMasterKey(os.Getenv(MasterKeyEnv))
}

// ParseMasterKey decodes and validates a 256-bit master key.
func ParseMasterKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("%s is required", MasterKeyEnv)
	}

	if b, err := base64.StdEncoding.DecodeString(value); err == nil && len(b) == fieldKeySize {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(value); err == nil && len(b) == fieldKeySize {
		return b, nil
	}
	if b, err := hex.DecodeString(value); err == nil && len(b) == fieldKeySize {
		return b, nil
	}
	if len([]byte(value)) == fieldKeySize {
		return []byte(value), nil
	}

	return nil, errors.New("master key must decode to exactly 32 bytes")
}

// DeriveFieldKey derives a stable AES-256 key for one logical field using
// HKDF-SHA256. Keeping keys per-field limits accidental key/nonce reuse across
// PHI domains and lets us rotate individual fields later.
func DeriveFieldKey(master []byte, field string) ([]byte, error) {
	if len(master) != fieldKeySize {
		return nil, errors.New("master key must be 32 bytes")
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return nil, errors.New("field name is required")
	}
	return hkdfSHA256(master, []byte("medsec-ocr-field-encryption-v1"), []byte(field), fieldKeySize), nil
}

func hkdfSHA256(secret, salt, info []byte, size int) []byte {
	if len(salt) == 0 {
		salt = make([]byte, sha256.Size)
	}

	extract := hmac.New(sha256.New, salt)
	extract.Write(secret)
	prk := extract.Sum(nil)

	var out []byte
	var previous []byte
	counter := byte(1)
	for len(out) < size {
		expand := hmac.New(sha256.New, prk)
		expand.Write(previous)
		expand.Write(info)
		expand.Write([]byte{counter})
		previous = expand.Sum(nil)
		out = append(out, previous...)
		counter++
	}
	return out[:size]
}
