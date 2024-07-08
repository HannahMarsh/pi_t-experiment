package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	pl "github.com/HannahMarsh/PrettyLogger"
	"golang.org/x/crypto/curve25519"
	"io"
	"math/big"
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

// GenerateECDHKeys generates a new ECDH key pair using X25519
func GenerateECDHKeys() ([]byte, []byte, error) {
	privKey := make([]byte, curve25519.ScalarSize)
	_, err := rand.Read(privKey)
	if err != nil {
		return nil, nil, err
	}

	// The base point for Curve25519 (9 in little-endian encoding), defined in the original Curve25519 paper by Daniel J. Bernstein.
	basepoint := []byte{
		9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	pubKey, err := curve25519.X25519(privKey, basepoint)
	if err != nil {
		return nil, nil, err
	}

	return privKey, pubKey, nil
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

// GenerateScalar generates a random scalar for use in the Diffie-Hellman exchange.
// Returns:
// - A byte slice representing the scalar.
// - An error object if an error occurred, otherwise nil.
func GenerateScalar() ([]byte, error) {
	scalar := make([]byte, 32) // 256-bit scalar
	if _, err := rand.Read(scalar); err != nil {
		return nil, err
	}
	return scalar, nil
}

// EncryptWithAES encrypts plaintext using AES encryption.
// Parameters:
// - key: The AES key.
// - plaintext: The plaintext to be encrypted.
// Returns:
// - The encrypted ciphertext.
// - The encrypted ciphertext (base64-encoded as string).
// - An error object if an error occurred, otherwise nil.
func EncryptWithAES(key, plaintext []byte) (cipherText []byte, encodedCiphertext string, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", err
	}

	cipherText = make([]byte, aes.BlockSize+len(plaintext))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv) // replace with ctr
	stream.XORKeyStream(cipherText[aes.BlockSize:], plaintext)

	encodedCiphertext = base64.StdEncoding.EncodeToString(cipherText)

	return cipherText, encodedCiphertext, nil
}

func EncryptStringWithAES(key []byte, plaintext string) (cipherText []byte, encodedCiphertext string, err error) {
	plainBytes, err := base64.StdEncoding.DecodeString(plaintext)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to decode plaintext")
	}
	return EncryptWithAES(key, plainBytes)
}

// DecryptWithAES decrypts ciphertext using AES encryption.
// Parameters:
// - key: The AES key.
// - ct: The base64-encoded ciphertext to be decrypted.
// Returns:
// - The decrypted plaintext.
// - The decrypted plaintext (base64-encoded as string).
// - An error object if an error occurred, otherwise nil.
func DecryptWithAES(key []byte, ciphertext []byte) (plainText []byte, encodedPlainText string, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to create new cipher")
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, "", pl.NewError("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	plainText = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(plainText, plainText)

	encodedPlainText = base64.StdEncoding.EncodeToString(plainText)
	return plainText, encodedPlainText, nil
}

func DecryptStringWithAES(key []byte, ct string) (plainText []byte, encodedPlainText string, err error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		return nil, "", pl.WrapError(err, "failed to decode ciphertext")
	}
	return DecryptWithAES(key, ciphertext)
}

// EncodeWithScalar encrypts the payload and returns the encrypted shared key and payload.
// Parameters:
// - payload: The plaintext payload to be encrypted.
// - privateKeyPEM: The PEM-encoded private key of the node.
// - publicKeyPEM: The PEM-encoded public key of the receiver.
// - scalar: The random scalar used in the Diffie-Hellman exchange.
// Returns:
// - The encrypted shared key.
// - The encrypted payload.
func EncodeWithScalar(payload []byte, privateKeyPEM string, publicKeyPEM string, scalar []byte) (encryptedSharedKey string, encryptedPayload string, err error) {
	symmetricKey, err := GenerateSymmetricKey()
	if err != nil {
		return "", "", pl.WrapError(err, "failed to generate symmetric key")
	}

	_, encryptedPayload, err = EncryptWithAES(symmetricKey, payload)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to encrypt payload")
	}

	sharedKey, err := ComputeSharedKeyWithScalar(privateKeyPEM, publicKeyPEM, scalar)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to compute shared key")
	}

	_, encryptedKey, err := EncryptWithAES(sharedKey, symmetricKey)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to encrypt key")
	}

	return base64.StdEncoding.EncodeToString([]byte(encryptedKey)), encryptedPayload, nil
}

// DecodeWithScalar decrypts the encrypted shared key and payload.
// Parameters:
// - encryptedSharedKey: The base64-encoded encrypted shared key.
// - encryptedPayload: The base64-encoded encrypted payload.
// - privateKeyPEM: The PEM-encoded private key of the node.
// - publicKeyPEM: The PEM-encoded public key of the sender.
// Returns:
// - The decrypted payload.
// - An error object if an error occurred, otherwise nil.
func DecodeWithScalar(encryptedSharedKey string, encryptedPayload string, privateKeyPEM string, publicKeyPEM string, scalar []byte) (decryptedPayload []byte, err error) {
	encryptedKey, err := base64.StdEncoding.DecodeString(encryptedSharedKey)
	if err != nil {
		return nil, err
	}

	sharedKey, err := ComputeSharedKeyWithScalar(privateKeyPEM, publicKeyPEM, scalar)
	if err != nil {
		return nil, err
	}

	symmetricKey, _, err := DecryptStringWithAES(sharedKey, string(encryptedKey))
	if err != nil {
		return nil, err
	}

	decryptedBytes, _, err := DecryptStringWithAES(symmetricKey, encryptedPayload)
	if err != nil {
		return nil, err
	}

	return decryptedBytes, nil
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

	_, encryptedPayload, err = EncryptWithAES(symmetricKey, payload)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to encrypt payload")
	}

	sharedKey, err := ComputeSharedKey(privateKeyPEM, publicKeyPEM)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to compute shared key")
	}

	_, encryptedKey, err := EncryptWithAES(sharedKey, symmetricKey)
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

	symmetricKey, _, err := DecryptStringWithAES(sharedKey, string(encryptedKey))
	if err != nil {
		return nil, err
	}

	decryptedBytes, _, err := DecryptStringWithAES(symmetricKey, encryptedPayload)
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
func decodePrivateKey(privateKeyPEM string) (*ecdh.PrivateKey, *ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, nil, pl.NewError("invalid private key PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to parse private key")
	}

	privKeyECDSA, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, nil, pl.NewError("failed to cast key to *ecdsa.PrivateKey")
	}

	privKey, err := privKeyECDSA.ECDH()
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to cast key to *ecdh.PrivateKey")
	}

	return privKey, privKeyECDSA, nil
}

// DecodePublicKey decodes the given PEM-encoded public key.
// Parameters:
// - publicKeyPEM: The PEM-encoded public key.
// Returns:
// - The decoded ECDH public key.
func DecodePublicKey(publicKeyPEM string) (*ecdh.PublicKey, *ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "EC PUBLIC KEY" {
		return nil, nil, pl.NewError("invalid public key PEM block")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to parse public key")
	}

	pubKeyECDSA, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, pl.NewError("failed to cast key to *ecdsa.ClientPubKey")
	}

	pubKey, err := pubKeyECDSA.ECDH()
	if err != nil {
		return nil, nil, pl.WrapError(err, "failed to cast key to *ecdh.ClientPubKey")
	}

	return pubKey, pubKeyECDSA, nil
}

// ComputeSharedKey computes the shared secret using the ECDH private key and a peer's public key.
// Parameters:
// - privKeyPEM: The PEM-encoded private key of the node.
// - pubKeyPEM: The PEM-encoded public key of the peer.
// Returns:
// - A byte slice representing the shared key.
// - An error object if an error occurred, otherwise nil.
func ComputeSharedKey(privKeyPEM, pubKeyPEM string) ([]byte, error) {
	privKey, _, err := decodePrivateKey(privKeyPEM)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode private key")
	}

	pubKey, _, err := DecodePublicKey(pubKeyPEM)
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

// ComputeSharedKeyWithScalar computes the shared secret using the ECDH private key and a peer's public key.
// Parameters:
// - privKeyPEM: The PEM-encoded private key of the node.
// - pubKeyPEM: The PEM-encoded public key of the peer.
// - scalar: The random scalar used in the Diffie-Hellman exchange.
// Returns:
// - A byte slice representing the shared key.
func ComputeSharedKeyWithScalar(privKeyPEM, pubKeyPEM string, scalar []byte) ([]byte, error) {
	_, decodedPrivKeyECDSA, err := decodePrivateKey(privKeyPEM)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode private key")
	}

	decodedPubKeyECDH, _, err := DecodePublicKey(pubKeyPEM)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode public key")
	}

	// Multiply the private key with the scalar
	scaledPrivKey, err := scalePrivateKey(decodedPrivKeyECDSA, scalar)
	if err != nil {
		return nil, pl.WrapError(err, "failed to scale private key with scalar")
	}

	sharedKey, err := scaledPrivKey.ECDH(decodedPubKeyECDH)
	if err != nil {
		return nil, pl.WrapError(err, "failed to compute shared key")
	}

	hashedSharedKey := sha256.Sum256(sharedKey)
	return hashedSharedKey[:], nil
}

// scalePrivateKey scales the given ECDH private key with the provided scalar.
// Parameters:
// - privKey: The ECDH private key.
// - scalar: The scalar to multiply with the private key.
// Returns:
// - The scaled ECDH private key.
func scalePrivateKey(privKeyECDSA *ecdsa.PrivateKey, scalar []byte) (*ecdh.PrivateKey, error) {
	curve := elliptic.P256()
	scalarBigInt := new(big.Int).SetBytes(scalar)

	// Multiply the private key's D value by the scalar
	scaledD := new(big.Int).Mul(privKeyECDSA.D, scalarBigInt)
	scaledD.Mod(scaledD, curve.Params().N)

	// Create a new ECDSA private key with the scaled D value
	scaledPrivKeyECDSA := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
		D: scaledD,
	}

	// Generate the public key from the scaled private key
	scaledPrivKeyECDSA.PublicKey.X, scaledPrivKeyECDSA.PublicKey.Y = curve.ScalarBaseMult(scaledD.Bytes())

	// Convert the scaled ECDSA private key back to an ECDH private key
	scaledECDHPrivKey, err := scaledPrivKeyECDSA.ECDH()
	if err != nil {
		return nil, err
	}

	return scaledECDHPrivKey, nil
}
