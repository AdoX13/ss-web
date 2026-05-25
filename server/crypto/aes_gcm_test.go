package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestAESGCM_NISTEmptyPlaintextVector(t *testing.T) {
	key := mustHex(t, "0000000000000000000000000000000000000000000000000000000000000000")
	nonce := mustHex(t, "000000000000000000000000")
	want := mustHex(t, "530f8afbc74536b9a963b4f1c4cb738b")

	got, err := sealAESGCM(key, nonce, nil, nil)
	if err != nil {
		t.Fatalf("sealAESGCM: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ciphertext+tag mismatch\nwant %x\n got %x", want, got)
	}

	plain, err := openAESGCM(key, nonce, got, nil)
	if err != nil {
		t.Fatalf("openAESGCM: %v", err)
	}
	if len(plain) != 0 {
		t.Fatalf("want empty plaintext, got %q", plain)
	}
}

func TestEncryptFieldRoundTrip(t *testing.T) {
	master := bytes.Repeat([]byte{0x42}, 32)
	env, err := EncryptString(master, "patient.name", "Maria Popescu")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	got, err := DecryptString(master, env)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if got != "Maria Popescu" {
		t.Fatalf("roundtrip mismatch: %q", got)
	}
}

func TestDecryptFieldRejectsTampering(t *testing.T) {
	master := bytes.Repeat([]byte{0x24}, 32)
	env, err := EncryptString(master, "patient.cnp", "1960101123456")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	env.Ciphertext[0] ^= 0xff
	if _, err := DecryptString(master, env); err == nil {
		t.Fatal("expected authentication error")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	return b
}
