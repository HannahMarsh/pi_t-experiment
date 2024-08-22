package pi_t

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	om "github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"strings"
)

// PeelOnion is responsible for peeling a layer off the onion and decrypting its contents.
// It returns the role of the current layer, the layer number, metadata, the peeled onion, the next destination, and an error (if any).
func PeelOnion(onion string, privateKey string) (role string, layer int, metadata *om.Metadata, peeled om.Onion, nextDestination string, err error) {

	// Decode the base64-encoded onion string into bytes.
	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode onion")
	}

	// Unmarshal the onion bytes into an Onion struct.
	var o om.Onion
	if err = json.Unmarshal(onionBytes, &o); err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to unmarshal onion")
	}

	// Decode the header using the private key to retrieve the ciphertext, next hop, and next header.
	cypherText, nextHop, nextHeader, err := o.Header.DecodeHeader(privateKey)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode header")
	}

	// Decode the layer key from the ciphertext.
	layerKey, err := base64.StdEncoding.DecodeString(cypherText.Key)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode layer key")
	}

	var decryptedContent om.Content

	// Determine if the current recipient is not a gatekeeper or mixer, and decrypt the content accordingly.
	if cypherText.Recipient != om.LAST_GATEKEEPER && cypherText.Recipient != om.GATEKEEPER && cypherText.Recipient != om.MIXER {
		decryptedContent, err = o.Content.DecryptContent(layerKey)
		if err != nil {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		}
		contentBytes, err := base64.StdEncoding.DecodeString(string(decryptedContent))
		if err != nil {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode content")
		}
		decryptedContent = om.Content(string(contentBytes))
		decryptedContent = om.Content(strings.TrimRight(string(contentBytes), "\x00")) // Remove padding from the decrypted content.
		layer = cypherText.Layer
		nextDestination = nextHop
		metadata = &cypherText.Metadata
		peeled = om.Onion{
			Header:  nextHeader,
			Sepal:   om.Sepal{}, // No sepal for non-mixer and non-gatekeeper layers.
			Content: decryptedContent,
		}
		return cypherText.Recipient, layer, metadata, peeled, nextDestination, nil
	}

	// If the current recipient is a gatekeeper or mixer, peel the sepal.
	peeledSepal, err := o.Sepal.PeelSepal(layerKey)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to peel sepal")
	}

	// If the recipient is the last gatekeeper, decrypt the content using the master key.
	if cypherText.Recipient == om.LAST_GATEKEEPER {
		masterKey := peeledSepal.Blocks[0]
		if masterKey == "" || masterKey == "null" {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "null master key, onion was likely bruised too many times")
		}
		K, err := base64.StdEncoding.DecodeString(masterKey)
		if err != nil {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode master key")
		}
		decryptedContent, err = o.Content.DecryptContent(K)
		if err != nil {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		}
	} else {
		// For other gatekeepers or mixers, decrypt the content using the layer key.
		decryptedContent, err = o.Content.DecryptContent(layerKey)
		if err != nil {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		}
	}

	// Set the layer number, next destination, and metadata.
	layer = cypherText.Layer
	nextDestination = nextHop
	metadata = &cypherText.Metadata
	peeled = om.Onion{
		Header:  nextHeader,
		Sepal:   peeledSepal, // Include the peeled sepal in the new onion.
		Content: decryptedContent,
	}

	// Return the recipient's role, layer number, metadata, peeled onion, and next destination.
	return cypherText.Recipient, layer, metadata, peeled, nextDestination, nil
}
