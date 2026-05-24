package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Argon2id parameters per OWASP recommendations.
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// HashPassword returns an Argon2id-encoded hash of password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword checks password against a stored hash. Handles both Argon2id
// (new) and bcrypt (legacy) formats, enabling transparent migration.
func VerifyPassword(password, hash string) (bool, error) {
	if strings.HasPrefix(hash, "$argon2id$") {
		return verifyArgon2id(password, hash)
	}
	// Legacy bcrypt: $2a$... or $2b$...
	if strings.HasPrefix(hash, "$2") {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err == nil {
			return true, nil
		}
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, fmt.Errorf("auth: bcrypt compare: %w", err)
	}
	return false, errors.New("auth: unrecognised hash format")
}

func verifyArgon2id(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	// $argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>  → 6 parts after split on $
	if len(parts) != 6 {
		return false, errors.New("auth: malformed argon2id hash")
	}
	var memory uint32
	var time, threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, fmt.Errorf("auth: parse argon2id params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("auth: decode salt: %w", err)
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("auth: decode hash: %w", err)
	}
	computed := argon2.IDKey([]byte(password), salt, uint32(time), memory, uint32(threads), uint32(len(storedHash)))
	return subtle.ConstantTimeCompare(computed, storedHash) == 1, nil
}
