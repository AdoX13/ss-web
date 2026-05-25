package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode"
)

// NormalizeCNP keeps only digits so equivalent OCR/entry forms hash the same.
func NormalizeCNP(cnp string) string {
	var b strings.Builder
	for _, r := range cnp {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// HashCNP returns a deterministic HMAC-SHA256 digest suitable for equality
// lookup and unique indexes without exposing the raw Romanian CNP.
func HashCNP(master []byte, cnp string) (string, error) {
	normalized := NormalizeCNP(cnp)
	if normalized == "" {
		return "", errors.New("cnp is required")
	}
	key, err := DeriveFieldKey(master, "cnp_lookup_hmac")
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(normalized))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
