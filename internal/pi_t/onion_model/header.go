package onion_model

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"strings"
)

type Header struct {
	E string
	B []string
	A []string // verification hashes
}

type CypherText struct {
	Tag       string
	Recipient string
	Layer     int
	Key       string
}

type CypherTextWrapper struct {
	Address    string
	CypherText string
}

func FormHeaders(l int, l1 int, C []Content, A [][]string, privateKey string, publicKeys []string, recipient string, layerKeys [][]byte, K []byte, path []string, hash func(string) string) (H []Header, err error) {

	// tag array
	tags := make([]string, l+1)
	tags[l] = hash(string(C[l]))

	// ciphertext array
	E := make([]string, l+1)
	E[l], err = enc(privateKey, publicKeys[l-1], tags[l], recipient, l, layerKeys[l])
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt ciphertext")
	}

	// header array
	H = make([]Header, l+1)
	H[l] = Header{
		E: E[l],
		B: []string{},
	}

	B := make([][]string, l+1)
	for i, _ := range B {
		B[i] = make([]string, l+1)
	}
	B[l-1][1], err = encryptB(recipient, E[l], K)
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt B_l_minus_1_1")
	}

	for i := l - 1; i >= 1; i-- {
		B[i][1], err = encryptB(path[i+1], E[i+1], layerKeys[i])
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt B_i_1")
		}
		for j := 2; j <= l-j+1; j++ {
			B[i][j], err = encryptB("", B[i+1][j-1], layerKeys[i])
		}
		B_i_1_to_C_i := append(B[i][1:], string(C[i]))
		concat := strings.Join(B_i_1_to_C_i, "")
		tags[i] = hash(concat)
		role := "mixer"
		if i == l-1 {
			role = "lastGatekeeper"
		} else if i > l1 {
			role = "gatekeeper"
		}
		E[i], err = enc(privateKey, publicKeys[i-1], tags[i], role, i, layerKeys[i]) // TODO add y_i, A_i
		if i-1 < len(A) {
			H[i] = Header{
				E: E[i],
				B: B[i],
				A: A[i-1],
			}
		} else {
			H[i] = Header{
				E: E[i],
				B: B[i],
			}
		}
	}

	return H, nil
}

func encryptB(address string, E string, layerKey []byte) (string, error) {
	b, err := json.Marshal(CypherTextWrapper{
		Address:    address,
		CypherText: E,
	})
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal b")
	}
	_, bEncrypted, err := keys.EncryptWithAES(layerKey, b)
	return bEncrypted, nil
}

func enc(privateKey, publicKey string, tag string, role string, layer int, layerKey []byte) (string, error) {
	sharedKey, err := keys.ComputeSharedKey(privateKey, publicKey)
	if err != nil {
		return "", pl.WrapError(err, "failed to compute shared key")
	}
	ciphertext := CypherText{
		Tag:       tag,
		Recipient: role,
		Layer:     layer,
		Key:       base64.StdEncoding.EncodeToString(layerKey),
	}
	cypherBytes, err := json.Marshal(ciphertext)
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal ciphertext")
	}

	_, E_l, err := keys.EncryptWithAES(sharedKey, cypherBytes)
	if err != nil {
		return "", pl.WrapError(err, "failed to encrypt ciphertext")
	}
	return E_l, nil
}

func dec(privateKey string, cyphertext string) (string, error) {

}
