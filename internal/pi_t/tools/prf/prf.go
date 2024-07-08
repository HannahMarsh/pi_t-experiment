package prf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"golang.org/x/exp/slog"
)

// PRF_F1 determines if a checkpoint onion is expected.
// Parameters:
// - privateKeyPEM: The PEM-encoded private key of either the node (at jth index of routing path) or the client forming the onion.
// - publicKeyPEM: The PEM-encoded public key of either the node (at jth index of routing path) or the client forming the onion.
// - j: The current hop index.
// Returns:
// - An integer, 0 or 1, indicating whether a checkpoint onion is expected.
func PRF_F1(privateKeyPEM, publicKeyPEM string, scalar []byte, j int) int {
	sharedKey, err := keys.ComputeSharedKeyWithScalar(privateKeyPEM, publicKeyPEM, scalar)
	if err != nil {
		slog.Error("failed to compute shared key", err)
		return 0
	}
	h := hmac.New(sha256.New, sharedKey)
	if err = binary.Write(h, binary.BigEndian, int64(j)); err != nil {
		slog.Error("failed to write to HMAC", err)
		return 0
	}
	result := h.Sum(nil)
	return int(result[0]) % 2 // Returns 0 or 1
}

// PRF_F2 computes the expected nonce for a given hop index.
// Parameters:
// - privateKeyPEM: The PEM-encoded private key of either the node (at jth index of routing path) or the client forming the onion.
// - publicKeyPEM: The PEM-encoded public key of either the node (at jth index of routing path) or the client forming the onion.
// - j: The current hop index.
// Returns:
// - A byte slice representing the expected nonce.
func PRF_F2(privateKeyPEM, publicKeyPEM string, scalar []byte, j int) []byte {
	sharedKey, err := keys.ComputeSharedKeyWithScalar(privateKeyPEM, publicKeyPEM, scalar)
	if err != nil {
		slog.Error("failed to compute shared key", err)
		return nil
	}
	h := hmac.New(sha256.New, sharedKey)
	if err = binary.Write(h, binary.BigEndian, int64(j)); err != nil {
		slog.Error("failed to write to HMAC", err)
		return nil
	}
	return h.Sum(nil)[:16] // Return the first 16 bytes as the nonce
}
