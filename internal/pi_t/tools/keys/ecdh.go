package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	pl "github.com/HannahMarsh/PrettyLogger"
	"golang.org/x/crypto/nacl/box"
	"io"
)

// KeyGen generates a [private, public] key pair using curve25519.
// Returns:
// - privateKeyHex: The hex-encoded private key.
// - publicKeyHex: The hex-encoded public key.
func KeyGen() (privateKeyHex string, publicKeyHex string, err error) {
	// curve := ecdh.P256() // Using P256 curve
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to generate ECDH key pair")
	}

	return hex.EncodeToString(privKey[:]), hex.EncodeToString(pubKey[:]), nil
}

// GenerateSymmetricKey generates a random AES key for encryption.
// Returns:
// - A byte slice representing the AES key.
// - An error object if an error occurred, otherwise nil.
func GenerateSymmetricKey() ([]byte, error) {
	key := make([]byte, 32) // AES-GCM
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptWithAES encrypts a message using AES-GCM with a shared secret key
func EncryptWithAES(sharedSecretKey, plaintext []byte) (cipherText []byte, encodedCiphertext string, err error) {
	block, err := aes.NewCipher(sharedSecretKey)
	if err != nil {
		return nil, "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, base64.StdEncoding.EncodeToString(ciphertext), nil
}

func EncryptStringWithAES(key []byte, plaintext string) (cipherText []byte, encodedCiphertext string, err error) {
	plainBytes, err := base64.StdEncoding.DecodeString(plaintext)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to decode plaintext")
	}
	//plainBytes := []byte(plaintext)
	return EncryptWithAES(key, plainBytes)
}

// DecryptWithAES decrypts a message using AES-GCM with a shared secret key
func DecryptWithAES(sharedSecretKey []byte, data []byte) (plainText []byte, encodedPlainText string, err error) {
	block, err := aes.NewCipher(sharedSecretKey)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to create new cipher")
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to create new GCM")
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, "", pl.WrapError(err, "ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plainText, err = aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to decrypt ciphertext")
	}

	return plainText, base64.StdEncoding.EncodeToString(plainText), nil
}

func DecryptStringWithAES(key []byte, ct string) (plainText []byte, encodedPlainText string, err error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to decode ciphertext")
	}
	//ciphertext := []byte(ct)
	return DecryptWithAES(key, ciphertext)
}

// DecodeHexKey decodes a hex-encoded string to a 32-byte array
func DecodeHexKey(hexKey string) ([32]byte, error) {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return [32]byte{}, err
	}

	var key [32]byte
	copy(key[:], bytes)
	return key, nil
}

func DecodeHexKeys(hexKey1 string, hexKey2 string) ([32]byte, [32]byte, error) {
	key1, err := DecodeHexKey(hexKey1)
	if err != nil {
		return [32]byte{}, [32]byte{}, pl.WrapError(err, "failed to decode hex key 1")
	}

	key2, err := DecodeHexKey(hexKey2)
	if err != nil {
		return [32]byte{}, [32]byte{}, pl.WrapError(err, "failed to decode hex key 2")
	}

	return key1, key2, nil
}

// ComputeSharedKey computes the shared secret using the ECDH private key and a peer's public key.
// Parameters:
// - privKeyHex: The hex-encoded private key of the node.
// - pubKeyHex: The hex-encoded public key of the peer.
// Returns:
// - A byte slice representing the shared key.
// - An error object if an error occurred, otherwise nil.
func ComputeSharedKey(privKeyHex, pubKeyHex string) ([32]byte, error) {

	privKey, pubKey, err := DecodeHexKeys(privKeyHex, pubKeyHex)
	if err != nil {
		return [32]byte{}, pl.WrapError(err, "failed to decode keys")
	}
	// Generate the shared key
	var sharedKey [32]byte
	box.Precompute(&sharedKey, &pubKey, &privKey)

	return sharedKey, nil
}
