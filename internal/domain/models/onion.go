package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

type Layer struct {
	EncryptedData []byte
	NextNode      Destination
	Key           []byte
}

type Destination struct {
	IP   string
	Port int
}

type Onion struct {
	ID     string
	Layers []Layer
}

// NewOnion creates a new onion
func NewOnion(id string) *Onion {
	return &Onion{
		ID: id,
	}
}

// AddLayer adds a new layer of encryption to the onion.
func (o *Onion) AddLayer(data []byte, nextNode Destination, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	newLayer := Layer{
		EncryptedData: ciphertext,
		NextNode:      nextNode,
		Key:           key,
	}
	o.Layers = append(o.Layers, newLayer)
	return nil
}

func (o *Onion) HasNextLayer() bool {
	return len(o.Layers) > 0
}

// RemoveLayer removes the outermost layer of encryption.
func (o *Onion) RemoveLayer() ([]byte, error) {
	if o.HasNextLayer() {
		return nil, errors.New("no layers left")
	}

	outerLayer := o.Layers[len(o.Layers)-1]
	o.Layers = o.Layers[:len(o.Layers)-1]

	block, err := aes.NewCipher(outerLayer.Key)
	if err != nil {
		return nil, err
	}

	if len(outerLayer.EncryptedData) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := outerLayer.EncryptedData[:aes.BlockSize]
	ciphertext := outerLayer.EncryptedData[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// GetNextNode retrieves the next node's information from the current layer.
func (o *Onion) GetNextNode() (Destination, error) {
	if len(o.Layers) == 0 {
		return Destination{}, errors.New("no layers left")
	}

	return o.Layers[len(o.Layers)-1].NextNode, nil
}
