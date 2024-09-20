package onion_model

import (
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
)

// Content is the encrypted content of an onion.
type Content string

// FormContent generates the encrypted content for each layer of the onion.
func FormContent(layerKeys [][]byte, l int, message []byte, K []byte) ([]Content, error) {
	C := make([]Content, l+1) // Initialize a slice to hold the encrypted content for each layer.

	// Encrypt the content for the last layer using the corresponding layer key.
	_, C_l, err := keys.EncryptWithAES(layerKeys[l], message)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l")
	}
	C[l] = Content(C_l) // Store the encrypted content for the last layer.

	// Encrypt the content for the second-to-last layer using the shared key K.
	_, C_l_misus_1, err := keys.EncryptStringWithAES(K, C_l)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l_minus_1")
	}
	C[l-1] = Content(C_l_misus_1) // Store the encrypted content for the second-to-last layer.

	// Encrypt the content for the remaining layers in reverse order.
	for i := l - 2; i >= 1; i-- {
		_, C_i, err := keys.EncryptStringWithAES(layerKeys[i], string(C[i+1]))
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt C_i")
		}
		C[i] = Content(C_i) // Store the encrypted content for the current layer.
	}
	return C, nil // Return the encrypted content for all layers.
}

// DecryptContent decrypts the encrypted content using the provided layer key.
func (c Content) DecryptContent(layerKey []byte) (Content, error) {
	_, decryptedString, err := keys.DecryptStringWithAES(layerKey, string(c))
	if err != nil {
		return "", pl.WrapError(err, "failed to decrypt content")
	}
	return Content(decryptedString), nil // Return the decrypted content.
}
