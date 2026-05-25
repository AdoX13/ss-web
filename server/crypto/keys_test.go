package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestParseMasterKeyAcceptsSupportedEncodings(t *testing.T) {
	raw := bytes.Repeat([]byte{0x7a}, 32)
	cases := []string{
		base64.StdEncoding.EncodeToString(raw),
		base64.RawStdEncoding.EncodeToString(raw),
		hex.EncodeToString(raw),
		string(raw),
	}
	for _, tc := range cases {
		got, err := ParseMasterKey(tc)
		if err != nil {
			t.Fatalf("ParseMasterKey(%q): %v", tc, err)
		}
		if !bytes.Equal(got, raw) {
			t.Fatalf("key mismatch for %q", tc)
		}
	}
}

func TestDeriveFieldKeySeparatesFields(t *testing.T) {
	master := bytes.Repeat([]byte{0x11}, 32)
	nameKey, err := DeriveFieldKey(master, "patients.name")
	if err != nil {
		t.Fatalf("DeriveFieldKey name: %v", err)
	}
	cnpKey, err := DeriveFieldKey(master, "patients.cnp")
	if err != nil {
		t.Fatalf("DeriveFieldKey cnp: %v", err)
	}
	if bytes.Equal(nameKey, cnpKey) {
		t.Fatal("different fields must not derive identical keys")
	}
}
