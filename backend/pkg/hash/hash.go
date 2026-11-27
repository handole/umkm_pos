package hash

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a raw password using bcrypt (cost factor 12).
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash compares a raw password against a bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashPIN produces a salted SHA-256 hash for quick POS PINs.
func HashPIN(pin, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(pin + ":" + salt))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CheckPINHash verifies a raw PIN against its salted hash.
func CheckPINHash(pin, salt, expectedHash string) bool {
	return HashPIN(pin, salt) == expectedHash
}
