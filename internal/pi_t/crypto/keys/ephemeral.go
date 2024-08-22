package keys

import (
	pl "github.com/HannahMarsh/PrettyLogger"
)

// GenerateEphemeralKeyPair generates an ephemeral key pair and returns the shared secret key computed with the relay's
// public key. The round number is embedded into the ephemeral public key to ensure it's tied to a specific session/round.
func GenerateEphemeralKeyPair(relayPublicKeyHex string) (ephemeralSharedSecret [32]byte, ephemeralPublicKeyHex string, err error) {

	var ephemeralPrivateKeyHex string

	ephemeralPrivateKeyHex, ephemeralPublicKeyHex, err = KeyGen()
	if err != nil {
		return [32]byte{}, "", pl.WrapError(err, "failed to generate ephemeral ECDH key pair")
	}
	secretKey, err := ComputeSharedKey(ephemeralPrivateKeyHex, relayPublicKeyHex)
	if err != nil {
		return [32]byte{}, "", pl.WrapError(err, "failed to compute shared key")
	}

	return secretKey, ephemeralPublicKeyHex, nil
}

func ComputeEphemeralSharedSecret(relayPrivateKeyHex, ephemeralPublicKeyHex string) ([32]byte, error) {

	sharedKey, err := ComputeSharedKey(relayPrivateKeyHex, ephemeralPublicKeyHex)
	if err != nil {
		return [32]byte{}, pl.WrapError(err, "failed to compute ephemeral shared secret")
	}

	return sharedKey, nil
}
