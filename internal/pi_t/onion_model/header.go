package onion_model

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
)

// Constants representing roles in the onion routing network.
const (
	GATEKEEPER      = "gatekeeper"
	MIXER           = "mixer"
	LAST_GATEKEEPER = "lastGatekeeper"
)

// Header represents the header of an onion, containing encryption-related metadata.
type Header struct {
	EPK        string   // Ephemeral public key.
	E          string   // Ciphertext encryption under the shared secret key computed with EPK.
	A          []string // Verification hashes.
	NextHeader string   // Encrypted next header under the layer key.
}

// CypherText represents the encrypted data for a specific layer in the onion.
type CypherText struct {
	Tag       string   // Tag used for verification.
	Recipient string   // Recipient's role (e.g., mixer, gatekeeper).
	Layer     int      // Layer number.
	Key       string   // Encryption key for the layer.
	Metadata  Metadata // Additional metadata for the layer.
}

// Metadata represents additional information associated with a layer.
type Metadata struct {
	Example string
	Nonce   string // Nonce used for encryption.
}

// CypherTextWrapper represents the next header information in the onion.
type CypherTextWrapper struct {
	Address    string // Address of the next node.
	NextHeader string // Encrypted next header.
}

// FormHeaders generates the headers for each lay erof the onion.
func FormHeaders(l int, l1 int, C []Content, A [][]string, publicKeys []string, recipient string, layerKeys [][]byte, path []string, hash func(string) string, metadata []Metadata) (H []Header, err error) {

	sharedSecrets := make([][32]byte, l+1)     // Shared secrets for each layer.
	publicEphemeralKeys := make([]string, l+1) // Ephemeral public keys for each layer.

	// Generate ephemeral keys and shared secrets for each layer.
	for i := 1; i <= l; i++ {
		sharedSecrets[i], publicEphemeralKeys[i], err = keys.GenerateEphemeralKeyPair(publicKeys[i-1])
		if err != nil {
			return nil, pl.WrapError(err, "failed to generate ephemeral key pair")
		}
	}

	// Initialize the tag array and generate the tag for the last layer.
	tags := make([]string, l+1)
	tags[l] = hash(string(C[l]))

	// Initialize the ciphertext array and encrypt the last layer's ciphertext.
	E := make([]string, l+1)
	E[l], err = enc(sharedSecrets[l], tags[l], recipient, l, layerKeys[l], metadata[l])
	if err != nil {
		return nil, pl.WrapError(err, "failed to encrypt ciphertext")
	}

	// Initialize the header array and set the header for the last layer.
	H = make([]Header, l+1)
	H[l] = Header{
		E:   E[l],
		EPK: publicEphemeralKeys[l],
	}

	// Generate headers for the remaining layers in reverse order.
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

		// Set the header for the current layer.
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

	return H, nil // Return the generated headers.
}

// encryptB encrypts the next header information for the given layer.
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

// enc encrypts the ciphertext for the given layer using the shared key.
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

// DecodeHeader decrypts and decodes the header using the provided private key.
func (h Header) DecodeHeader(privateKey string) (*CypherText, string, Header, error) {
	// Compute the shared secret key using the ephemeral public key (EPK) and the relay's private key.
	sharedKey, err := keys.ComputeEphemeralSharedSecret(privateKey, h.EPK)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to compute secret key")
	}

	// Decrypt the ciphertext in the header.
	cypherbytes, _, err := keys.DecryptStringWithAES(sharedKey[:], h.E)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decrypt ciphertext")
	}

	// Unmarshal the decrypted data into a CypherText struct.
	var ciphertext CypherText
	err = json.Unmarshal(cypherbytes, &ciphertext)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to unmarshal ciphertext")
	}

	// Decode the layer key from the base64-encoded string.
	layerKey, err := base64.StdEncoding.DecodeString(ciphertext.Key)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decode layer key")
	}

	// If there is no next header, return the decoded ciphertext.
	if h.NextHeader == "" {
		return &ciphertext, "", Header{}, nil
	}

	// Decrypt the next header using the layer key.
	nextHeader, _, err := keys.DecryptStringWithAES(layerKey, h.NextHeader)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decrypt next header")
	}
	var ctw CypherTextWrapper
	err = json.Unmarshal(nextHeader, &ctw)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to unmarshal next header")
	}

	// Decode the next header from the base64-encoded string.
	nextHeaderBytes, err := base64.StdEncoding.DecodeString(ctw.NextHeader)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to decode next header")
	}

	// Unmarshal the next header.
	var nh Header
	err = json.Unmarshal(nextHeaderBytes, &nh)
	if err != nil {
		return nil, "", Header{}, pl.WrapError(err, "failed to unmarshal next header")
	}

	return &ciphertext, ctw.Address, nh, nil // Return the decoded ciphertext and next header information.
}
