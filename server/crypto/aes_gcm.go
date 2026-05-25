package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	AlgorithmAES256GCM = "AES-256-GCM-HKDF-SHA256"
	envelopeVersion    = 1
	gcmNonceSize       = 12
)

// Envelope is the BSON/JSON shape stored for one encrypted PHI field.
type Envelope struct {
	Algorithm  string `json:"alg" bson:"alg"`
	Version    int    `json:"v" bson:"v"`
	Field      string `json:"field" bson:"field"`
	Nonce      []byte `json:"nonce" bson:"nonce"`
	Ciphertext []byte `json:"ciphertext" bson:"ciphertext"`
}

// EncryptField encrypts plaintext using a per-field AES-256-GCM key derived
// from master. The field name is authenticated as AAD.
func EncryptField(master []byte, field string, plaintext []byte) (*Envelope, error) {
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}
	return encryptFieldWithNonce(master, field, plaintext, nonce)
}

// DecryptField authenticates and decrypts one encrypted PHI field.
func DecryptField(master []byte, env *Envelope) ([]byte, error) {
	if env == nil {
		return nil, errors.New("crypto: nil envelope")
	}
	if env.Algorithm != AlgorithmAES256GCM {
		return nil, fmt.Errorf("crypto: unsupported algorithm %q", env.Algorithm)
	}
	if env.Version != envelopeVersion {
		return nil, fmt.Errorf("crypto: unsupported envelope version %d", env.Version)
	}
	if len(env.Nonce) != gcmNonceSize {
		return nil, errors.New("crypto: invalid nonce length")
	}

	key, err := DeriveFieldKey(master, env.Field)
	if err != nil {
		return nil, err
	}
	return openAESGCM(key, env.Nonce, env.Ciphertext, aad(env.Field))
}

func EncryptString(master []byte, field, plaintext string) (*Envelope, error) {
	return EncryptField(master, field, []byte(plaintext))
}

func DecryptString(master []byte, env *Envelope) (string, error) {
	plaintext, err := DecryptField(master, env)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func encryptFieldWithNonce(master []byte, field string, plaintext, nonce []byte) (*Envelope, error) {
	if len(nonce) != gcmNonceSize {
		return nil, errors.New("crypto: invalid nonce length")
	}
	key, err := DeriveFieldKey(master, field)
	if err != nil {
		return nil, err
	}
	ciphertext, err := sealAESGCM(key, nonce, plaintext, aad(field))
	if err != nil {
		return nil, err
	}
	return &Envelope{
		Algorithm:  AlgorithmAES256GCM,
		Version:    envelopeVersion,
		Field:      field,
		Nonce:      append([]byte(nil), nonce...),
		Ciphertext: ciphertext,
	}, nil
}

func aad(field string) []byte {
	return []byte(fmt.Sprintf("%s:%d:%s", AlgorithmAES256GCM, envelopeVersion, field))
}

func sealAESGCM(key, nonce, plaintext, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
	}
	return gcm.Seal(nil, nonce, plaintext, additionalData), nil
}

func openAESGCM(key, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("crypto: authenticate/decrypt: %w", err)
	}
	return plaintext, nil
}
