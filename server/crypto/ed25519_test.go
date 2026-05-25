package crypto

import (
	"encoding/hex"
	"testing"
)

func TestEd25519SignVerify(t *testing.T) {
	public, private, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("GenerateEd25519Key: %v", err)
	}
	payload := []byte("evidence payload")
	sig, err := SignEd25519(private, payload)
	if err != nil {
		t.Fatalf("SignEd25519: %v", err)
	}
	if !VerifyEd25519(public, payload, sig) {
		t.Fatal("signature did not verify")
	}
	if VerifyEd25519(public, []byte("tampered"), sig) {
		t.Fatal("tampered payload verified")
	}
}

func TestParseEd25519PrivateKey(t *testing.T) {
	_, private, err := GenerateEd25519Key()
	if err != nil {
		t.Fatalf("GenerateEd25519Key: %v", err)
	}
	encoded := EncodeEd25519PrivateKey(private)
	parsed, err := ParseEd25519PrivateKey(encoded)
	if err != nil {
		t.Fatalf("ParseEd25519PrivateKey: %v", err)
	}
	public, err := PublicKeyFromPrivate(parsed)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivate: %v", err)
	}
	sig, _ := SignEd25519(parsed, []byte("payload"))
	if !VerifyEd25519(public, []byte("payload"), sig) {
		t.Fatal("parsed key signature failed")
	}
}

func TestParseEd25519PrivateKeyAcceptsHexSeed(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	parsed, err := ParseEd25519PrivateKey(hex.EncodeToString(seed))
	if err != nil {
		t.Fatalf("ParseEd25519PrivateKey hex seed: %v", err)
	}
	public, err := PublicKeyFromPrivate(parsed)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivate: %v", err)
	}
	sig, _ := SignEd25519(parsed, []byte("payload"))
	if !VerifyEd25519(public, []byte("payload"), sig) {
		t.Fatal("hex seed signature failed")
	}
}
