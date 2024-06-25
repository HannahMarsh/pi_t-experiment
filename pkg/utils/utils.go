package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// ref: https://www.thorsten-hans.com/check-if-application-is-running-in-docker-container/
func IsRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}

	return true
}

func GenerateUniqueHash() (string, error) {
	// Create a buffer for random data
	randomData := make([]byte, 32) // 32 bytes for a strong unique value

	// Fill the buffer with random data
	if _, err := io.ReadFull(rand.Reader, randomData); err != nil {
		return "", err
	}

	// Create a new SHA256 hash
	hash := sha256.New()

	// Write the random data to the hash
	hash.Write(randomData)

	// Get the resulting hash as a byte slice
	hashBytes := hash.Sum(nil)

	// Encode the hash to a hexadecimal string
	hashString := hex.EncodeToString(hashBytes)

	return hashString, nil
}
