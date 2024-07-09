package pi_t

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	om "github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"strings"
)

func PeelOnion(onion string, sharedKey [32]byte) (layer int, metadata *om.Metadata, peeled om.Onion, nextDestination string, err error) {

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode onion")
	}
	var o om.Onion
	if err = json.Unmarshal(onionBytes, &o); err != nil {
		return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to unmarshal onion")
	}
	cypherText, nextHop, nextHeader, err := o.Header.DecodeHeader(sharedKey)
	if err != nil {
		return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode header")
	}

	layerKey, err := base64.StdEncoding.DecodeString(cypherText.Key)

	var decryptedContent om.Content

	if cypherText.Recipient != "lastGatekeeper" && cypherText.Recipient != "gatekeeper" && cypherText.Recipient != "mixer" {
		decryptedContent, err = o.Content.DecryptContent(layerKey)
		if err != nil {
			return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		}
		contentBytes, err := base64.StdEncoding.DecodeString(string(decryptedContent))
		if err != nil {
			return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode content")
		}
		decryptedContent = om.Content(string(contentBytes))
		decryptedContent = om.Content(strings.TrimRight(string(contentBytes), "\x00"))
		decryptedContent = om.Content(base64.StdEncoding.EncodeToString([]byte(decryptedContent)))
		layer = cypherText.Layer
		nextDestination = nextHop
		metadata = &cypherText.Metadata
		peeled = om.Onion{
			Header:  nextHeader,
			Sepal:   om.Sepal{},
			Content: decryptedContent,
		}
		return layer, metadata, peeled, nextDestination, nil
	}
	peeledSepal, err := o.Sepal.PeelSepal(layerKey)
	if err != nil {
		return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to peel sepal")
	}

	if cypherText.Recipient == "lastGatekeeper" {
		masterKey := peeledSepal.Blocks[0]
		K, err := base64.StdEncoding.DecodeString(masterKey)
		if err != nil {
			return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decode master key")
		}
		decryptedContent, err = o.Content.DecryptContent(K)
		if err != nil {
			return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		}
		//
		//result, _, err := keys.DecryptStringWithAES(K, nextHeader.E)
		//if err != nil {
		//	return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt with AES")
		//}
		//slog.Info("", "", result)
		//decryptedContent, err = o.Content.DecryptContent(K)
		//if err != nil {
		//	return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
		//}

	} else {
		decryptedContent, err = o.Content.DecryptContent(layerKey)
		if err != nil {
			return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
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

	return layer, metadata, peeled, nextDestination, nil
}

func BruiseOnion(onion string, privateKeyPEM string) {

}
