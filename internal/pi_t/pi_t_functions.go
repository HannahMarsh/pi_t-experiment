package pi_t

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"strings"

	"github.com/HannahMarsh/PrettyLogger"
)

// KeyGen generates an RSA key pair and returns the public and private keys in PEM format
func KeyGen() (privateKeyPEM, publicKeyPEM string, err error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", PrettyLogger.WrapError(err, "failed to generate private key")
	}

	// Encode private key to PEM format
	privateKeyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	privateKeyPEM = string(privateKeyPEMBytes)

	// Generate public key
	publicKey := &privateKey.PublicKey

	// Encode public key to PEM format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", "", PrettyLogger.WrapError(err, "failed to marshal public key")
	}
	publicKeyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	publicKeyPEM = string(publicKeyPEMBytes)

	return privateKeyPEM, publicKeyPEM, nil
}

// GenerateSymmetricKey generates a random AES key for encryption
func GenerateSymmetricKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptWithAES encrypts plaintext using AES encryption
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

// DecryptWithAES decrypts ciphertext using AES encryption
func DecryptWithAES(key []byte, ct string) ([]byte, error) {
	ciphertext, _ := base64.StdEncoding.DecodeString(ct)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

type OnionLayer struct {
	NextHop string
	Payload string
}

// FormOnion creates an onion by encapsulating a message in multiple encryption layers
func FormOnion(payload []byte, publicKeys []string, routingPath []string) (string, string, error) {
	for i := len(publicKeys) - 1; i >= 0; i-- {
		var layerBytes []byte
		var err error
		if i == len(publicKeys)-1 {
			layerBytes = payload
		} else {
			layer := OnionLayer{
				NextHop: routingPath[i+1],
				Payload: base64.StdEncoding.EncodeToString(payload),
			}

			layerBytes, err = json.Marshal(layer)
			if err != nil {
				return "", "", err
			}
		}

		symmetricKey, err := GenerateSymmetricKey()
		if err != nil {
			return "", "", err
		}

		encryptedPayload, err := EncryptWithAES(symmetricKey, layerBytes)
		if err != nil {
			return "", "", err
		}

		pubKeyBlock, _ := pem.Decode([]byte(publicKeys[i]))
		if pubKeyBlock == nil || pubKeyBlock.Type != "RSA PUBLIC KEY" {
			return "", "", errors.New("invalid public key PEM block")
		}

		pubKey, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
		if err != nil {
			return "", "", err
		}

		encryptedKey, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey.(*rsa.PublicKey), symmetricKey)
		if err != nil {
			return "", "", err
		}

		combinedPayload := struct {
			Key     string
			Payload string
		}{
			Key:     base64.StdEncoding.EncodeToString(encryptedKey),
			Payload: encryptedPayload,
		}

		payload, err = json.Marshal(combinedPayload)
		if err != nil {
			return "", "", err
		}
	}

	return routingPath[0], base64.StdEncoding.EncodeToString(payload), nil
}

// PeelOnion removes the outermost layer of the onion
func PeelOnion(onion string, privateKeyPEM string) (string, string, error) {
	privateKeyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	if privateKeyBlock == nil || privateKeyBlock.Type != "RSA PRIVATE KEY" {
		return "", "", errors.New("invalid private key PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return "", "", err
	}

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return "", "", err
	}

	var combinedPayload struct {
		Key     string
		Payload string
	}
	if err = json.Unmarshal(onionBytes, &combinedPayload); err != nil {
		return "", "", err
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(combinedPayload.Key)
	if err != nil {
		return "", "", err
	}

	symmetricKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedKey)
	if err != nil {
		return "", "", err
	}

	decryptedBytes, err := DecryptWithAES(symmetricKey, combinedPayload.Payload)
	if err != nil {
		return "", "", err
	}

	if !strings.HasPrefix(string(decryptedBytes), "{\"NextHop\":") {
		return "", string(decryptedBytes), nil
	}

	var layer OnionLayer
	err = json.Unmarshal(decryptedBytes, &layer)
	if err != nil {
		return "", string(decryptedBytes), nil
	}

	return layer.NextHop, layer.Payload, nil
}

// BruiseOnion modifies the onion payload to introduce bruising
func BruiseOnion(onion string) (string, error) {
	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return "", err
	}

	// Introduce bruising by modifying a small portion of the payload
	if len(onionBytes) > 0 {
		onionBytes[0] ^= 0xFF
	} else {
		return "", errors.New("empty onion")
	}

	return base64.StdEncoding.EncodeToString(onionBytes), nil
}
