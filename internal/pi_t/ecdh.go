package pi_t

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	pl "github.com/HannahMarsh/PrettyLogger"
	"io"
	"strings"
)

// GenerateECDHKeyPair generates an ECDH key pair using the P256 curve
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

func decodePrivateKey(privateKeyPEM string, publicKeyPEM string) (*ecdh.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, pl.NewError("invalid private key PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, pl.WrapError(err, "failed to parse private key")
	}

	privKeyECDSA, ok := key.(*ecdh.PrivateKey)
	if !ok {
		return nil, pl.NewError("failed to cast key to *ecdsa.PrivateKey")
	}

	pubKey, err := decodePublicKey(publicKeyPEM)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode public key")
	}

	privKey, err := privKeyECDSA.ECDH(pubKey)
	if err != nil {
		return nil, pl.WrapError(err, "failed to convert to ECDH private key")
	}

	return privKey, nil
}

func decodePublicKey(publicKeyPEM string) (*ecdh.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "EC PUBLIC KEY" {
		return nil, pl.NewError("invalid public key PEM block")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, pl.WrapError(err, "failed to parse public key")
	}

	pubKey, ok := key.(*ecdh.PublicKey)
	if !ok {
		return nil, pl.NewError("failed to cast key to *ecdh.PublicKey")
	}

	return pubKey, nil
}

// ComputeSharedKey computes the shared secret using the ECDH private key and a peer's public key
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
		return nil, pl.NewError("ciphertext too short")
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

type Onion struct {
	IsCheckpointOnion bool
	Layer             int
	NextHop           string
	Payload           string
}

// FormOnion creates an onion by encapsulating a message in multiple encryption layers
func FormOnion(privateKeyPEM string, publicKeyPEM string, payload []byte, publicKeys []string, routingPath []string, checkpoint int) (string, string, error) {
	for i := len(publicKeys) - 1; i >= 0; i-- {
		var layerBytes []byte
		var err error
		if i == len(publicKeys)-1 {
			layerBytes = payload
		} else {
			layer := Onion{
				IsCheckpointOnion: checkpoint == i,
				Layer:             i,
				NextHop:           routingPath[i+1],
				Payload:           base64.StdEncoding.EncodeToString(payload),
			}

			layerBytes, err = json.Marshal(layer)
			if err != nil {
				return "", "", pl.WrapError(err, "failed to marshal onion layer")
			}
		}

		symmetricKey, err := GenerateSymmetricKey()
		if err != nil {
			return "", "", pl.WrapError(err, "failed to generate symmetric key")
		}

		encryptedPayload, err := EncryptWithAES(symmetricKey, layerBytes)
		if err != nil {
			return "", "", pl.WrapError(err, "failed to encrypt payload")
		}

		sharedKey, err := ComputeSharedKey(privateKeyPEM, publicKeys[i])
		if err != nil {
			return "", "", pl.WrapError(err, "failed to compute shared key")
		}

		encryptedKey, err := EncryptWithAES(sharedKey, symmetricKey)
		if err != nil {
			return "", "", pl.WrapError(err, "failed to encrypt key")
		}

		combinedPayload := struct {
			Key       string
			Payload   string
			PublicKey string
		}{
			Key:       base64.StdEncoding.EncodeToString([]byte(encryptedKey)),
			Payload:   encryptedPayload,
			PublicKey: publicKeyPEM,
		}

		payload, err = json.Marshal(combinedPayload)
		if err != nil {
			return "", "", pl.WrapError(err, "failed to marshal combined payload")
		}
	}

	return routingPath[0], base64.StdEncoding.EncodeToString(payload), nil
}

// PeelOnion removes the outermost layer of the onion
func PeelOnion(onion string, privateKeyPEM string) (*Onion, error) {
	//privateKey, err := decodePrivateKey(privateKeyPEM)
	//if err != nil {
	//	return nil, err
	//}

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return nil, err
	}

	var combinedPayload struct {
		Key       string
		Payload   string
		PublicKey string
	}
	if err = json.Unmarshal(onionBytes, &combinedPayload); err != nil {
		return nil, err
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(combinedPayload.Key)
	if err != nil {
		return nil, err
	}

	sharedKey, err := ComputeSharedKey(privateKeyPEM, combinedPayload.PublicKey)
	if err != nil {
		return nil, err
	}

	symmetricKey, err := DecryptWithAES(sharedKey, string(encryptedKey))
	if err != nil {
		return nil, err
	}

	decryptedBytes, err := DecryptWithAES(symmetricKey, combinedPayload.Payload)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(string(decryptedBytes), "{\"IsCheckpointOnion\":") {
		return &Onion{
			IsCheckpointOnion: false,
			Layer:             0,
			NextHop:           "",
			Payload:           string(decryptedBytes),
		}, nil
	}

	var layer Onion
	err = json.Unmarshal(decryptedBytes, &layer)
	if err != nil {
		return nil, err
	}
	return &layer, nil
}

//// GenerateECDHKeyPair generates an ECDH key pair using the P256 curve
//func GenerateECDHKeyPair() (privateKeyPEM string, publicKeyPEM string, err error) {
//	var privKey *ecdh.PrivateKey
//
//	curve := ecdh.P256() // Using P256 curve
//	if privKey, err = curve.GenerateKey(rand.Reader); err != nil {
//		return "", "", pl.WrapError(err, "failed to generate ECDH key pair")
//	} else {
//		if privateKeyPEM, publicKeyPEM, err = encodeKeys(privKey); err != nil {
//			return "", "", pl.WrapError(err, "failed to encode keys")
//		} else {
//			return privateKeyPEM, publicKeyPEM, nil
//		}
//	}
//}
//
//func encodeKeys(privKey *ecdh.PrivateKey) (privateKeyPEM string, publicKeyPEM string, err error) {
//	var publicKeyBytes, privateKeyBytes []byte
//	if publicKeyBytes, err = x509.MarshalPKIXPublicKey(privKey.PublicKey()); err != nil {
//		return "", "", pl.WrapError(err, "failed to marshal public key")
//	} else if privateKeyBytes, err = x509.MarshalPKCS8PrivateKey(privKey); err != nil {
//		return "", "", pl.WrapError(err, "failed to marshal private key")
//	} else {
//		publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
//			Type:  "EC PUBLIC KEY",
//			Bytes: publicKeyBytes,
//		}))
//		privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
//			Type:  "EC PRIVATE KEY",
//			Bytes: privateKeyBytes,
//		}))
//
//		return privateKeyPEM, publicKeyPEM, nil
//	}
//}
//
//func decodePrivateKey(privateKeyPEM string) (privKey *ecdh.PrivateKey, err error) {
//	var privateKeyBytes *pem.Block
//	var key any
//	var ok bool
//	if privateKeyBytes, _ = pem.Decode([]byte(privateKeyPEM)); privateKeyBytes.Type != "EC PRIVATE KEY" {
//		return nil, pl.NewError("invalid private key PEM block. Wrong type: %s", privateKeyBytes.Type)
//	} else if key, err = x509.ParsePKCS8PrivateKey(privateKeyBytes.Bytes); err != nil {
//		return nil, pl.WrapError(err, "failed to parse private key")
//	} else if privKey, ok = key.(*ecdh.PrivateKey); !ok {
//		return nil, pl.NewError("failed to cast key to *ecdh.PrivateKey")
//	} else {
//		return privKey, nil
//	}
//}
//
//func decodePublicKey(publicKeyPEM string) (pubKey *ecdh.PublicKey, err error) {
//	var publicKeyBytes *pem.Block
//	var key any
//	var ok bool
//	if publicKeyBytes, _ = pem.Decode([]byte(publicKeyPEM)); publicKeyBytes.Type != "EC PUBLIC KEY" {
//		return nil, pl.NewError("invalid public key PEM block. Wrong type: %s", publicKeyBytes.Type)
//	} else if key, err = x509.ParsePKIXPublicKey(publicKeyBytes.Bytes); err != nil {
//		return nil, pl.WrapError(err, "failed to parse public key")
//	} else if pubKey, ok = key.(*ecdh.PublicKey); !ok {
//		return nil, pl.NewError("failed to cast key to *ecdh.PublicKey")
//	} else {
//		return pubKey, nil
//	}
//}
//
//// ComputeSharedKey computes the shared secret using the ECDH private key and a peer's public key
//func ComputeSharedKey(privKeyPEM, pubKeyPEM string) ([]byte, error) {
//	var privKey *ecdh.PrivateKey
//	var sharedKey []byte
//
//	if pubKey, err := decodePublicKey(pubKeyPEM); err != nil {
//		return nil, pl.WrapError(err, "failed to decode public key")
//	} else if privKey, err = decodePrivateKey(privKeyPEM); err != nil {
//		return nil, pl.WrapError(err, "failed to decode private key")
//	} else if sharedKey, err = privKey.ECDH(pubKey); err != nil {
//		return nil, pl.WrapError(err, "failed to compute shared key")
//	} else {
//		hashedSharedKey := sha256.Sum256(sharedKey)
//		return hashedSharedKey[:], err
//	}
//}
