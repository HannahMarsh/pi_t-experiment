package onion_model

import (
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
)

type Content string

func FormContent(layerKeys [][]byte, l int, message []byte, K []byte) ([]Content, error) {
	C := make([]Content, l+1)

	_, C_l, err := keys.EncryptWithAES(layerKeys[l], message)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l")
	}
	C[l] = Content(C_l)

	_, C_l_misus_1, err := keys.EncryptStringWithAES(K, C_l)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l_minus_1")
	}
	C[l-1] = Content(C_l_misus_1)

	for i := l - 2; i >= 1; i-- {
		_, C_i, err := keys.EncryptStringWithAES(layerKeys[i], string(C[i+1]))
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt C_i")
		}
		C[i] = Content(C_i)
	}
	return C, nil
}

func (c Content) DecryptContent(layerKey []byte) (Content, error) {
	_, decryptedString, err := keys.DecryptStringWithAES(layerKey, string(c))
	if err != nil {
		return "", pl.WrapError(err, "failed to decrypt content")
	}
	return Content(decryptedString), nil
}
