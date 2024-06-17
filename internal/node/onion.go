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
			return fmt.Errorf("failed to decrypt data: %v", err)
		} else if on, err2 := fromBytes(inner); err2 != nil {
			return fmt.Errorf("failed to decrypt data: %v", err2)
		} else {
			o.Address = on.Address
			o.Data = on.Data
			return nil
		}
	} else if o.HasMessage() {
		if message, err := decrypt(o.Message, privateKey); err != nil {
			return fmt.Errorf("failed to decrypt message: %v", err)
		} else {
			o.Message = message
			return nil
		}
	} else {
		return nil
	}
}

func (o *Onion) AddLayer(addr string, publicKey []byte) error {
	if bytes, err := toBytes(o); err != nil {
		return fmt.Errorf("failed to add layer: %v", err)
	} else if encryptedData, err2 := encrypt(bytes, publicKey); err2 != nil {
		return fmt.Errorf("failed to add layer: %v", err2)
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
		return nil, fmt.Errorf("failed to encode onion: %v", err)
	}
	return buf.Bytes(), nil
}

func fromBytes(data []byte) (*Onion, error) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	var o Onion
	if err := dec.Decode(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

func decrypt(data []byte, privateKey []byte) ([]byte, error) {
	if block, _ := pem.Decode(privateKey); block == nil {
		return nil, errors.New("failed to parse private key")
	} else if privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	} else if result, err2 := rsa.DecryptPKCS1v15(rand.Reader, privKey, data); err2 != nil { // Decrypt address and data
		return nil, fmt.Errorf("failed to decrypt address: %v", err2)
	} else {
		return result, nil
	}
}

func encrypt(data []byte, publicKey []byte) ([]byte, error) {
	if block, _ := pem.Decode(publicKey); block == nil {
		return nil, errors.New("failed to parse public key")
	} else if pubKey, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	} else if rsaPubKey, ok := pubKey.(*rsa.PublicKey); !ok {
		return nil, errors.New("failed to parse RSA public key")
	} else if encryptedData, err2 := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, data); err2 != nil {
		return nil, fmt.Errorf("failed to encrypt address: %v", err2)
	} else {
		return encryptedData, nil
	}
}
