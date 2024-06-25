package utils

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"github.com/HannahMarsh/PrettyLogger"
)

// GenerateECDHKeyPair generates a private/public key pair for ECDH using P256 curve.
func GenerateECDHKeyPair() (*ecdh.PrivateKey, *ecdh.PublicKey, error) {
	curve := ecdh.P256() // Using P256 curve
	if privKey, err := curve.GenerateKey(rand.Reader); err != nil {
		return nil, nil, PrettyLogger.WrapError(err, "failed to generate ECDH key pair")
	} else {
		return privKey, privKey.PublicKey(), nil
	}
}

// ComputeSharedKey computes the shared secret using the ECDH private key and a peer's public key.
func ComputeSharedKey(privKey *ecdh.PrivateKey, pubKey *ecdh.PublicKey) ([]byte, error) {
	if sharedKey, err := privKey.ECDH(pubKey); err != nil {
		return nil, PrettyLogger.WrapError(err, "failed to compute shared key")
	} else {
		hashedSharedKey := sha256.Sum256(sharedKey)
		return hashedSharedKey[:], nil
	}
}
