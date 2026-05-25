package crypto

import (
	"bytes"
	"testing"
)

func TestHashCNPDeterministicAndNormalized(t *testing.T) {
	master := bytes.Repeat([]byte{0x11}, 32)
	a, err := HashCNP(master, "1960101123456")
	if err != nil {
		t.Fatalf("HashCNP: %v", err)
	}
	b, err := HashCNP(master, "196 0101-123456")
	if err != nil {
		t.Fatalf("HashCNP normalized: %v", err)
	}
	if a != b {
		t.Fatalf("expected normalized hashes to match: %s != %s", a, b)
	}
}

func TestHashCNPDifferentMaster(t *testing.T) {
	a, err := HashCNP(bytes.Repeat([]byte{0x11}, 32), "1960101123456")
	if err != nil {
		t.Fatalf("HashCNP a: %v", err)
	}
	b, err := HashCNP(bytes.Repeat([]byte{0x22}, 32), "1960101123456")
	if err != nil {
		t.Fatalf("HashCNP b: %v", err)
	}
	if a == b {
		t.Fatal("different master keys must produce different HMACs")
	}
}
