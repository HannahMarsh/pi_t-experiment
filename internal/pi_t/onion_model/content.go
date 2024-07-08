package onion_model

import (
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
)

type Content string

func FormContent(layerKeys [][]byte, l int, message []byte, K []byte) ([]Content, error) {
	C_arr := make([]Content, l+1)
	C_l, _, err := keys.EncryptWithAES(layerKeys[l], message)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l")
	}
	_, C_el_encrypted, err := keys.EncryptWithAES(K, C_l)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt C_l_minus_1")
	}
	C_arr[l] = Content(C_el_encrypted)
	for i := l - 1; i >= 1; i-- {
		_, c_i_encrypted, err := keys.EncryptStringWithAES(layerKeys[i], string(C_arr[i+1]))
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt C_i")
		}
		C_arr[i] = Content(c_i_encrypted)
	}
	return C_arr, nil
}
