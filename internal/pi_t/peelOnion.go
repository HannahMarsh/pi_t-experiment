package pi_t

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	om "github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"strings"
)

func PeelOnion(onion string, privateKey string) (role string, layer int, metadata *om.Metadata, peeled om.Onion, nextDestination string, err error) {

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode onion")
	}
	var o om.Onion
	if err = json.Unmarshal(onionBytes, &o); err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to unmarshal onion")
	}
	cypherText, nextHop, nextHeader, err := o.Header.DecodeHeader(privateKey)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode header")
	}

	layerKey, err := base64.StdEncoding.DecodeString(cypherText.Key)

	var decryptedContent om.Content

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
		decryptedContent = om.Content(strings.TrimRight(string(contentBytes), "\x00"))
		layer = cypherText.Layer
		nextDestination = nextHop
		metadata = &cypherText.Metadata
		peeled = om.Onion{
			Header:  nextHeader,
			Sepal:   om.Sepal{},
			Content: decryptedContent,
		}
		return cypherText.Recipient, layer, metadata, peeled, nextDestination, nil
	}
	peeledSepal, err := o.Sepal.PeelSepal(layerKey)
	if err != nil {
		return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to peel sepal")
	}

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
		decryptedContent, err = o.Content.DecryptContent(layerKey)
		if err != nil {
			return "", -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		}
	}

	layer = cypherText.Layer
	nextDestination = nextHop
	metadata = &cypherText.Metadata
	peeled = om.Onion{
		Header:  nextHeader,
		Sepal:   peeledSepal,
		Content: decryptedContent,
	}

	return cypherText.Recipient, layer, metadata, peeled, nextDestination, nil
}