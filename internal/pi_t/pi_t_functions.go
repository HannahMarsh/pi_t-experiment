package pi_t

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
)

// KeyGen generates an RSA key pair and returns the public and private keys in PEM format
func KeyGen() (privateKeyPEM, publicKeyPEM string, err error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
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
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	publicKeyPEM = string(publicKeyPEMBytes)

	return privateKeyPEM, publicKeyPEM, nil
}

type OnionLayer struct {
	NextHop string
	Payload string
}

// FormOnion creates an onion by encapsulating a message in multiple encryption layers
func FormOnion(payload []byte, publicKeys []string, routingPath []string) (string, string, error) {

	for i := len(publicKeys) - 1; i >= 0; i-- {
		layer := OnionLayer{
			NextHop: routingPath[i],
			Payload: base64.StdEncoding.EncodeToString(payload),
		}

		layerBytes, err := json.Marshal(layer)
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

		payload, err = rsa.EncryptPKCS1v15(rand.Reader, pubKey.(*rsa.PublicKey), layerBytes)
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

	decryptedBytes, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, onionBytes)
	if err != nil {
		return "", "", err
	}

	var layer OnionLayer
	err = json.Unmarshal(decryptedBytes, &layer)
	if err != nil {
		return "", "", err
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
	}

	return base64.StdEncoding.EncodeToString(onionBytes), nil
}
