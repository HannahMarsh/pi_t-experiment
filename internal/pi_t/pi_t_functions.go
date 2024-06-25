package pi_t

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
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

type OnionPayload struct {
	IsCheckpointOnion bool
	Layer             int
	LastHop           string
	NextHop           string
	Payload           string
	NextHopPubKey     string
}

type CombinedPayload struct {
	EncryptedSharedKey   string
	EncryptedPayload     string
	OriginalSenderPubKey string
}

type Header struct {
	BruiseCounter            int
	EncryptedSharedKey       string
	SenderPubKey             string
	CombinedEncryptedPayload string
}

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

// FormOnion creates an onion by encapsulating a message in multiple encryption layers
func FormOnion(privateKeyPEM string, publicKeyPEM string, payload []byte, publicKeys []string, routingPath []string, checkpoint int) (string, string, error) {
	for i := len(publicKeys) - 1; i >= 0; i-- {
		var layerBytes []byte
		var err error
		if i == len(publicKeys)-1 {
			layerBytes = payload
		} else {
			lastHop := ""
			if i > 0 {
				lastHop = routingPath[i-1]
			}
			layer := OnionPayload{
				IsCheckpointOnion: checkpoint == i,
				Layer:             i,
				LastHop:           lastHop,
				NextHop:           routingPath[i+1],
				Payload:           base64.StdEncoding.EncodeToString(payload),
				NextHopPubKey:     publicKeys[i+1],
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

		combinedPayload := CombinedPayload{
			EncryptedSharedKey:   base64.StdEncoding.EncodeToString([]byte(encryptedKey)),
			EncryptedPayload:     encryptedPayload,
			OriginalSenderPubKey: publicKeyPEM,
		}

		payload, err = json.Marshal(combinedPayload)
		if err != nil {
			return "", "", pl.WrapError(err, "failed to marshal combined payload")
		}
	}

	onionWithHeader, err := addHeaderAfterPeeling(base64.StdEncoding.EncodeToString(payload), privateKeyPEM, publicKeyPEM, publicKeys[0], 0)
	if err != nil {
		return "", "", pl.WrapError(err, "failed to add header")
	}

	return routingPath[0], onionWithHeader, nil
}

func AddHeader(peeledOnion *OnionPayload, bruiseCounter int, privateKeyPEM string, senderPubicKey string) (string, error) {
	return addHeaderAfterPeeling(peeledOnion.Payload, privateKeyPEM, senderPubicKey, peeledOnion.NextHopPubKey, bruiseCounter)
}

func addHeaderAfterPeeling(payload string, privateKeyPEM string, senderPubicKey string, receiverPublicKey string, bruiseCounter int) (string, error) {

	payloadbytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	encryptedSharedKey, encryptedPayload, err := Enc(payloadbytes, privateKeyPEM, receiverPublicKey)
	if err != nil {
		return "", err
	}
	header := Header{
		BruiseCounter:            bruiseCounter,
		EncryptedSharedKey:       encryptedSharedKey,
		SenderPubKey:             senderPubicKey,
		CombinedEncryptedPayload: encryptedPayload,
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(headerBytes), nil
}

func removeHeader(onion string, privateKeyPEM string) (string, int, error) {
	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return "", 0, err
	}
	var header Header
	if err = json.Unmarshal(onionBytes, &header); err != nil {
		return "", 0, err
	}
	decryptedPayload, err := Dec(header.EncryptedSharedKey, header.CombinedEncryptedPayload, privateKeyPEM, header.SenderPubKey)
	if err != nil {
		return "", 0, err
	}
	return base64.StdEncoding.EncodeToString(decryptedPayload), header.BruiseCounter, nil
}

// PeelOnion removes the outermost layer of the onion
func PeelOnion(onion string, privateKeyPEM string) (*OnionPayload, int, error) {
	headerRemoved, bruises, err := removeHeader(onion, privateKeyPEM)
	if err != nil {
		return nil, -1, err
	}

	peeled, err := peelOnionAfterRemovingPayload(headerRemoved, privateKeyPEM)
	if err != nil {
		return nil, -1, err
	}

	return peeled, bruises, nil
}

// PeelOnion removes the outermost layer of the onion
func peelOnionAfterRemovingPayload(onion string, privateKeyPEM string) (*OnionPayload, error) {

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return nil, err
	}

	var combinedPayload CombinedPayload
	if err = json.Unmarshal(onionBytes, &combinedPayload); err != nil {
		return nil, err
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(combinedPayload.EncryptedSharedKey)
	if err != nil {
		return nil, err
	}

	sharedKey, err := ComputeSharedKey(privateKeyPEM, combinedPayload.OriginalSenderPubKey)
	if err != nil {
		return nil, err
	}

	symmetricKey, err := DecryptWithAES(sharedKey, string(encryptedKey))
	if err != nil {
		return nil, err
	}

	decryptedBytes, err := DecryptWithAES(symmetricKey, combinedPayload.EncryptedPayload)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(string(decryptedBytes), "{\"IsCheckpointOnion\":") {
		return &OnionPayload{
			IsCheckpointOnion: false,
			Layer:             0,
			NextHop:           "",
			LastHop:           "",
			Payload:           string(decryptedBytes),
		}, nil
	}

	var layer OnionPayload
	err = json.Unmarshal(decryptedBytes, &layer)
	if err != nil {
		return nil, err
	}
	return &layer, nil
}
