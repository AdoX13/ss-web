package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const EvidencePrivateKeyEnv = "EVIDENCE_ED25519_PRIVATE_KEY"

func GenerateEd25519Key() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func SignEd25519(private ed25519.PrivateKey, payload []byte) ([]byte, error) {
	if l := len(private); l != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("ed25519 private key must be %d bytes, got %d", ed25519.PrivateKeySize, l)
	}
	return ed25519.Sign(private, payload), nil
}

func VerifyEd25519(public ed25519.PublicKey, payload, signature []byte) bool {
	return len(public) == ed25519.PublicKeySize && ed25519.Verify(public, payload, signature)
}

func PublicKeyFromPrivate(private ed25519.PrivateKey) (ed25519.PublicKey, error) {
	if l := len(private); l != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("ed25519 private key must be %d bytes, got %d", ed25519.PrivateKeySize, l)
	}
	pub, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("ed25519 public key assertion failed")
	}
	return pub, nil
}

func LoadEd25519PrivateKeyFromEnv() (ed25519.PrivateKey, error) {
	return ParseEd25519PrivateKey(os.Getenv(EvidencePrivateKeyEnv))
}

// ParseEd25519PrivateKey accepts base64 or hex encoded private keys. A 32-byte
// seed is expanded into a full 64-byte private key.
func ParseEd25519PrivateKey(value string) (ed25519.PrivateKey, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("%s is required", EvidencePrivateKeyEnv)
	}

	var lastErr error
	for _, decode := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		hex.DecodeString,
	} {
		b, err := decode(value)
		if err != nil {
			lastErr = err
			continue
		}
		key, err := privateKeyFromBytes(b)
		if err == nil {
			return key, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("ed25519 private key must be base64 or hex")
}

func privateKeyFromBytes(b []byte) (ed25519.PrivateKey, error) {
	switch len(b) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(b), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(b), nil
	default:
		return nil, fmt.Errorf("ed25519 key must be %d-byte seed or %d-byte private key", ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}

func EncodeEd25519PrivateKey(private ed25519.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(private)
}

func EncodeEd25519PublicKey(public ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(public)
}
