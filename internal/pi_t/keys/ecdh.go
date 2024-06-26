package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	pl "github.com/HannahMarsh/PrettyLogger"
	"io"
)

// KeyGen generates an ECDH key pair using the P256 curve.
// Returns:
// - privateKeyPEM: The PEM-encoded private key.
// - publicKeyPEM: The PEM-encoded public key.
// - err: An error object if an error occurred, otherwise nil.
func KeyGen() (privateKeyPEM string, publicKeyPEM string, err error) {
	curve := ecdh.P256() // Using P256 curve
	privKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to generate ECDH key pair")
	}

	privateKeyPEM, publicKeyPEM, err = encodeKeys(privKey)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to encode keys")
	}

	return privateKeyPEM, publicKeyPEM, nil
}

// GenerateSymmetricKey generates a random AES key for encryption.
// Returns:
// - A byte slice representing the AES key.
// - An error object if an error occurred, otherwise nil.
func GenerateSymmetricKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptWithAES encrypts plaintext using AES encryption.
// Parameters:
// - key: The AES key.
// - plaintext: The plaintext to be encrypted.
// Returns:
// - The base64-encoded ciphertext.
// - An error object if an error occurred, otherwise nil.
func EncryptWithAES(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithAES decrypts ciphertext using AES encryption.
// Parameters:
// - key: The AES key.
// - ct: The base64-encoded ciphertext to be decrypted.
// Returns:
// - The decrypted plaintext.
// - An error object if an error occurred, otherwise nil.
func DecryptWithAES(key []byte, ct string) ([]byte, error) {
	ciphertext, _ := base64.StdEncoding.DecodeString(ct)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, pl.NewError("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// Enc encrypts the payload and returns the encrypted shared key and payload.
// Parameters:
// - payload: The plaintext payload to be encrypted.
// - privateKeyPEM: The PEM-encoded private key of the node.
// - publicKeyPEM: The PEM-encoded public key of the receiver.
// Returns:
// - The encrypted shared key.
// - The encrypted payload.
// - An error object if an error occurred, otherwise nil.
func Enc(payload []byte, privateKeyPEM string, publicKeyPEM string) (encryptedSharedKey string, encryptedPayload string, err error) {
	symmetricKey, err := GenerateSymmetricKey()
	if err != nil {
		return "", "", pl.WrapError(err, "failed to generate symmetric key")
	}

	encryptedPayload, err = EncryptWithAES(symmetricKey, payload)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to encrypt payload")
	}

	sharedKey, err := ComputeSharedKey(privateKeyPEM, publicKeyPEM)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to compute shared key")
	}

	encryptedKey, err := EncryptWithAES(sharedKey, symmetricKey)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to encrypt key")
	}

	return base64.StdEncoding.EncodeToString([]byte(encryptedKey)), encryptedPayload, nil
}

// Dec decrypts the encrypted shared key and payload.
// Parameters:
// - encryptedSharedKey: The base64-encoded encrypted shared key.
// - encryptedPayload: The base64-encoded encrypted payload.
// - privateKeyPEM: The PEM-encoded private key of the node.
// - publicKeyPEM: The PEM-encoded public key of the sender.
// Returns:
// - The decrypted payload.
// - An error object if an error occurred, otherwise nil.
func Dec(encryptedSharedKey string, encryptedPayload string, privateKeyPEM string, publicKeyPEM string) (decryptedPayload []byte, err error) {
	encryptedKey, err := base64.StdEncoding.DecodeString(encryptedSharedKey)
	if err != nil {
		return nil, err
	}

	sharedKey, err := ComputeSharedKey(privateKeyPEM, publicKeyPEM)
	if err != nil {
		return nil, err
	}

	symmetricKey, err := DecryptWithAES(sharedKey, string(encryptedKey))
	if err != nil {
		return nil, err
	}

	decryptedBytes, err := DecryptWithAES(symmetricKey, encryptedPayload)
	if err != nil {
		return nil, err
	}

	return decryptedBytes, nil
}

// encodeKeys encodes the given ECDH private key into PEM format.
// Parameters:
// - privKey: The ECDH private key.
// Returns:
// - privateKeyPEM: The PEM-encoded private key.
// - publicKeyPEM: The PEM-encoded public key.
// - err: An error object if an error occurred, otherwise nil.
func encodeKeys(privKey *ecdh.PrivateKey) (privateKeyPEM string, publicKeyPEM string, err error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(privKey.Public())
	if err != nil {
		return "", "", pl.WrapError(err, "failed to marshal public key")
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to marshal private key")
	}

	publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PUBLIC KEY",
		Bytes: publicKeyBytes,
	}))

	privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	return privateKeyPEM, publicKeyPEM, nil
}

// decodePrivateKey decodes the given PEM-encoded private key.
// Parameters:
// - privateKeyPEM: The PEM-encoded private key.
// Returns:
// - The decoded ECDH private key.
// - An error object if an error occurred, otherwise nil.
func decodePrivateKey(privateKeyPEM string) (*ecdh.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, pl.NewError("invalid private key PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, pl.WrapError(err, "failed to parse private key")
	}

	privKeyECDSA, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, pl.NewError("failed to cast key to *ecdsa.PrivateKey")
	}

	privKey, err := privKeyECDSA.ECDH()
	if err != nil {
		return nil, pl.WrapError(err, "failed to cast key to *ecdh.PrivateKey")
	}

	return privKey, nil
}

// decodePublicKey decodes the given PEM-encoded public key.
// Parameters:
// - publicKeyPEM: The PEM-encoded public key.
// Returns:
// - The decoded ECDH public key.
// - An error object if an error occurred, otherwise nil.
func decodePublicKey(publicKeyPEM string) (*ecdh.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "EC PUBLIC KEY" {
		return nil, pl.NewError("invalid public key PEM block")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, pl.WrapError(err, "failed to parse public key")
	}

	pubKeyECDSA, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, pl.NewError("failed to cast key to *ecdsa.OriginalSenderPubKey")
	}

	pubKey, err := pubKeyECDSA.ECDH()
	if err != nil {
		return nil, pl.WrapError(err, "failed to cast key to *ecdh.OriginalSenderPubKey")
	}

	return pubKey, nil
}

// ComputeSharedKey computes the shared secret using the ECDH private key and a peer's public key.
// Parameters:
// - privKeyPEM: The PEM-encoded private key of the node.
// - pubKeyPEM: The PEM-encoded public key of the peer.
// Returns:
// - A byte slice representing the shared key.
// - An error object if an error occurred, otherwise nil.
func ComputeSharedKey(privKeyPEM, pubKeyPEM string) ([]byte, error) {
	privKey, err := decodePrivateKey(privKeyPEM)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode private key")
	}

	pubKey, err := decodePublicKey(pubKeyPEM)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode public key")
	}

	sharedKey, err := privKey.ECDH(pubKey)
	if err != nil {
		return nil, pl.WrapError(err, "failed to compute shared key")
	}

	hashedSharedKey := sha256.Sum256(sharedKey)
	return hashedSharedKey[:], nil
}
