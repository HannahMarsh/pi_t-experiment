package pi_t

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	om "github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
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

	peeledSepal, err := o.Sepal.PeelSepal(layerKey, false)
	if err != nil {
		return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to peel sepal")
	}

	decryptedContent, err := o.Content.DecryptContent(layerKey)
	if err != nil {
		return -1, nil, om.Onion{}, "", pl.WrapError(err, "failed to decrypt content")
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
