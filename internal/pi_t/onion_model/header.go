package onion_model

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
)

type Header struct {
	EPK        string   // ephemeral public key
	E          string   // CypherText encryption under the shared secret key computed with EPK
	A          []string // verification hashes
	NextHeader string   // encryption under the layerKey of CypherTextWrapper
}

const (
	GATEKEEPER      = "gatekeeper"
	MIXER           = "mixer"
	LAST_GATEKEEPER = "lastGatekeeper"
)

type CypherText struct {
	Tag       string
	Recipient string
	Layer     int
	Key       string
	Metadata  Metadata
}

type Metadata struct {
	Example string
	Nonce   string
}

type CypherTextWrapper struct {
	Address    string
	NextHeader string
}

func FormHeaders(l int, l1 int, C []Content, A [][]string, publicKeys []string, recipient string, layerKeys [][]byte, path []string, hash func(string) string, metadata []Metadata) (H []Header, err error) {

	sharedSecrets := make([][32]byte, l+1)
	publicEphemeralKeys := make([]string, l+1)

	// generate ephemeral keys and shared secrets
	for i := 1; i <= l; i++ {
		sharedSecrets[i], publicEphemeralKeys[i], err = keys.GenerateEphemeralKeyPair(publicKeys[i-1])
		if err != nil {
			return nil, pl.WrapError(err, "failed to generate ephemeral key pair")
		}
	}

	// tag array
	tags := make([]string, l+1)
	tags[l] = hash(string(C[l]))

	// ciphertext array
	E := make([]string, l+1)
	E[l], err = enc(sharedSecrets[l], tags[l], recipient, l, layerKeys[l], metadata[l])
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt ciphertext")
	}

	// header array
	H = make([]Header, l+1)
	H[l] = Header{
		E:   E[l],
		EPK: publicEphemeralKeys[l],
	}

	for i := l - 1; i >= 1; i-- {
		role := MIXER
		if i == l-1 {
			role = LAST_GATEKEEPER
		} else if i > l1 {
			role = GATEKEEPER
		}
		E[i], err = enc(sharedSecrets[i], tags[i], role, i, layerKeys[i], metadata[i])
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
				EPK:        publicEphemeralKeys[i],
			}
		} else {
			H[i] = Header{
				E:          E[i],
				NextHeader: nh,
				EPK:        publicEphemeralKeys[i],
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

func enc(sharedKey [32]byte, tag string, role string, layer int, layerKey []byte, metadata Metadata) (string, error) {
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

func (h Header) DecodeHeader(privateKey string) (*CypherText, string, Header, error) {
	sharedKey, err := keys.ComputeEphemeralSharedSecret(privateKey, h.EPK)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to compute secret key")
	}

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
}
