package node

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"fmt"
)

type Onion struct {
	Address string
	Data    []byte
	Message []byte
}

func (o *Onion) IsDummy() bool {
	return o.Data == nil && o.Message == nil
}

func (o *Onion) HasNextLayer() bool {
	return o.Data != nil
}

func (o *Onion) HasMessage() bool {
	return o.Message != nil
}

func (o *Onion) RemoveLayer(privateKey []byte) error {
	// Parse private key
	if o.HasNextLayer() {
		if inner, err := decrypt(o.Data, privateKey); err != nil {
			return fmt.Errorf("onion.RemoveLayer(): failed to decrypt data: %w", err)
		} else if on, err2 := fromBytes(inner); err2 != nil {
			return fmt.Errorf("onion.RemoveLayer(): failed to decrypt data: %w", err2)
		} else {
			o.Address = on.Address
			o.Data = on.Data
			return nil
		}
	} else if o.HasMessage() {
		if message, err := decrypt(o.Message, privateKey); err != nil {
			return fmt.Errorf("onion.RemoveLayer(): failed to decrypt message: %w", err)
		} else {
			o.Message = message
			return nil
		}
	} else {
		return nil
	}
}

func (o *Onion) AddLayer(addr string, publicKey []byte) error {
	if b, err := toBytes(o); err != nil {
		return fmt.Errorf("onion.AddLayer(): failed to add layer: %w", err)
	} else if encryptedData, err2 := encrypt(b, publicKey); err2 != nil {
		return fmt.Errorf("onion.AddLayer(): failed to add layer: %w", err2)
	} else {
		o.Address = addr
		o.Data = encryptedData
		return nil
	}
}

func toBytes(o *Onion) ([]byte, error) {
	var buf bytes.Buffer        // Stand-in for a buf connection
	enc := gob.NewEncoder(&buf) // Will write to buf.
	// Encode (send) the value.
	if err := enc.Encode(o); err != nil {
		return nil, fmt.Errorf("toBytes(): failed to encode onion: %w", err)
	}
	return buf.Bytes(), nil
}

func fromBytes(data []byte) (*Onion, error) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	var o Onion
	if err := dec.Decode(&o); err != nil {
		return nil, fmt.Errorf("fromBytes(): failed to decode onion: %w", err)
	}
	return &o, nil
}

func decrypt(data []byte, privateKey []byte) ([]byte, error) {
	if block, _ := pem.Decode(privateKey); block == nil {
		return nil, fmt.Errorf("decrypt(): failed to parse private key: %s", string(privateKey))
	} else if privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("decrypt(): failed to parse private key: %w", err)
	} else if result, err2 := rsa.DecryptPKCS1v15(rand.Reader, privKey, data); err2 != nil { // Decrypt address and data
		return nil, fmt.Errorf("decrypt(): failed to decrypt address: %w", err2)
	} else {
		return result, nil
	}
}

func encrypt(data []byte, publicKey []byte) ([]byte, error) {
	if block, _ := pem.Decode(publicKey); block == nil {
		return nil, errors.New("encrypt(): failed to parse public key")
	} else if pubKey, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("encrypt(): failed to parse public key: %w", err)
	} else if rsaPubKey, ok := pubKey.(*rsa.PublicKey); !ok {
		return nil, errors.New("encrypt(): failed to parse RSA public key")
	} else if encryptedData, err2 := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, data); err2 != nil {
		return nil, fmt.Errorf("encrypt(): failed to encrypt address: %w", err2)
	} else {
		return encryptedData, nil
	}
}
