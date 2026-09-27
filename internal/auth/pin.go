package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"regexp"
)

// pbkdf2Iterations and pbkdf2KeyLen follow the spec's minimum of 210,000
// iterations of PBKDF2-SHA256, with a 32-byte derived key.
const (
	pbkdf2Iterations = 210_000
	pbkdf2KeyLen     = 32
	saltLen          = 16
)

var pinFormat = regexp.MustCompile(`^[0-9]{4,6}$`)

// ValidatePINFormat reports whether pin is 4 to 6 decimal digits, returning a
// human-readable error naming the problem when it is not.
func ValidatePINFormat(pin string) error {
	if !pinFormat.MatchString(pin) {
		return fmt.Errorf("PIN must be 4 to 6 digits")
	}
	return nil
}

// HashPIN derives a PBKDF2-SHA256 hash of pin with a fresh random salt,
// returning both as base64 for storage in users.json.
func HashPIN(pin string) (hash, salt string, err error) {
	saltBytes := make([]byte, saltLen)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, pin, saltBytes, pbkdf2Iterations, pbkdf2KeyLen)
	if err != nil {
		return "", "", fmt.Errorf("derive PIN hash: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), base64.StdEncoding.EncodeToString(saltBytes), nil
}

// VerifyPIN reports whether pin matches the stored hash for the given salt,
// both base64-encoded as produced by HashPIN. It runs in constant time.
func VerifyPIN(pin, hash, salt string) bool {
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}
	wantHash, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	gotHash, err := pbkdf2.Key(sha256.New, pin, saltBytes, pbkdf2Iterations, pbkdf2KeyLen)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(gotHash, wantHash) == 1
}
