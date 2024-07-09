package onion_model

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
)

type Header struct {
	E          string   // encryption under pk(Pi) of CypherText
	A          []string // verification hashes
	NextHeader string   // encryption under the layerKey of CypherTextWrapper
}

type CypherText struct {
	Tag       string
	Recipient string
	Layer     int
	Key       string
	Metadata  Metadata
}

type Metadata struct {
	Example string
	Nonce   int
}

type CypherTextWrapper struct {
	Address    string
	NextHeader string
}

func FormHeaders(l int, l1 int, C []Content, A [][]string, privateKey string, publicKeys []string, recipient string, layerKeys [][]byte, K []byte, path []string, hash func(string) string, metadata []Metadata) (H []Header, err error) {

	// tag array
	tags := make([]string, l+1)
	tags[l] = hash(string(C[l]))

	// ciphertext array
	E := make([]string, l+1)
	E[l], err = enc(privateKey, publicKeys[l-1], tags[l], recipient, l, layerKeys[l], metadata[l])
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt ciphertext")
	}

	// header array
	H = make([]Header, l+1)
	H[l] = Header{
		E: E[l],
	}

	//B := make([][]string, l+1)
	//for i, _ := range B {
	//	B[i] = make([]string, l+1)
	//}
	//B[l][1], err = encryptB(recipient, E[l], K)
	//if err != nil {
	//	return nil, pl.WrapError(err, "failed to encrypt B_l_minus_1_1")
	//}

	for i := l - 1; i >= 1; i-- {
		//B[i][1], err = encryptB(path[i+1], E[i+1], layerKeys[i])
		//if err != nil {
		//	return nil, pl.WrapError(err, "failed to encrypt B_i_1")
		//}
		//for j := 2; j <= l-j+1; j++ {
		//	B[i][j], err = encryptB("", B[i+1][j-1], layerKeys[i])
		//	if err != nil {
		//		return nil, pl.WrapError(err, "failed to encrypt B_i_j")
		//	}
		//}
		//B_i_1_to_C_i := append(B[i][1:], string(C[i]))
		//concat := strings.Join(B_i_1_to_C_i, "")
		//tags[i] = hash(concat)
		role := "mixer"
		if i == l-1 {
			role = "lastGatekeeper"
		} else if i > l1 {
			role = "gatekeeper"
		}
		E[i], err = enc(privateKey, publicKeys[i-1], tags[i], role, i, layerKeys[i], metadata[i])
		nextHeader := H[i+1]
		headerBytes, err := json.Marshal(nextHeader)
		if err != nil {
			return nil, pl.WrapError(err, "failed to marshal next header")
		}
		nh, err := encryptB(path[i+1], base64.StdEncoding.EncodeToString(headerBytes), layerKeys[i])
		if err != nil {
			return nil, pl.WrapError(err, "failed to encrypt next header")
		}

		if i-1 < len(A) {
			H[i] = Header{
				E:          E[i],
				A:          A[i-1],
				NextHeader: nh,
			}
		} else {
			H[i] = Header{
				E:          E[i],
				NextHeader: nh,
			}
		}
	}

	return H, nil
}

func encryptB(address string, nextHeader string, layerKey []byte) (string, error) {
	b, err := json.Marshal(CypherTextWrapper{
		Address:    address,
		NextHeader: nextHeader,
	})
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal b")
	}
	_, bEncrypted, err := keys.EncryptWithAES(layerKey, b)
	return bEncrypted, nil
}

func enc(privateKey, publicKey string, tag string, role string, layer int, layerKey []byte, metadata Metadata) (string, error) {
	sharedKey, err := keys.ComputeSharedKey(privateKey, publicKey)
	if err != nil {
		return "", pl.WrapError(err, "failed to compute shared key")
	}
	ciphertext := CypherText{
		Tag:       tag,
		Recipient: role,
		Layer:     layer,
		Key:       base64.StdEncoding.EncodeToString(layerKey),
		Metadata:  metadata,
	}
	cypherBytes, err := json.Marshal(ciphertext)
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal ciphertext")
	}

	_, E_l, err := keys.EncryptWithAES(sharedKey[:], cypherBytes)
	if err != nil {
		return "", pl.WrapError(err, "failed to encrypt ciphertext")
	}

	return E_l, nil
}

func (h Header) DecodeHeader(sharedKey [32]byte) (*CypherText, string, Header, error) {

	cypherbytes, _, err := keys.DecryptStringWithAES(sharedKey[:], h.E)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decrypt ciphertext")
	}

	var ciphertext CypherText
	err = json.Unmarshal(cypherbytes, &ciphertext)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to unmarshal ciphertext")
	}

	layerKey, err := base64.StdEncoding.DecodeString(ciphertext.Key)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decode layer key")
	}

	if h.NextHeader == "" {
		return &ciphertext, "", Header{}, nil
	}

	nextHeader, _, err := keys.DecryptStringWithAES(layerKey, h.NextHeader)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decrypt next header")
	}
	var ctw CypherTextWrapper
	err = json.Unmarshal(nextHeader, &ctw)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to unmarshal next header")
	}

	nextHeaderBytes, err := base64.StdEncoding.DecodeString(ctw.NextHeader)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decode next header")
	}

	var nh Header
	err = json.Unmarshal(nextHeaderBytes, &nh)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to unmarshal next header")
	}

	return &ciphertext, ctw.Address, nh, nil
	//
	//
	//ctwArr := utils.Map(h.B[1:], func(b string) CypherTextWrapper {
	//	if b != "" {
	//		b_bytes, _, err2 := keys.DecryptStringWithAES(layerKey, b)
	//		if err2 != nil {
	//			err = pl.WrapError(err2, "failed to decrypt b")
	//		}
	//		var ctw CypherTextWrapper
	//		err2 = json.Unmarshal(b_bytes, &ctw)
	//		if err2 != nil {
	//			err = pl.WrapError(err2, "failed to unmarshal b")
	//		}
	//		return ctw
	//	} else {
	//		return CypherTextWrapper{}
	//	}
	//})
	//if err != nil {
	//	return nil, nil, pl.WrapError(err, "failed to decrypt b")
	//}
	//
	////result, _, err := keys.DecryptStringWithAES(layerKey, ctwArr[0].NextHeader)
	////if err != nil {
	////	return nil, nil, pl.WrapError(err, "failed to decrypt next header")
	////}
	////slog.Info("", "", result)
	//
	//return &ciphertext, ctwArr, nil
}
