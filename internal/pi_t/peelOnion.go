package pi_t

import (
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	om "github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
)

func PeelOnion(onion string, privateKeyPEM string) (*om.Onion, error) {

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return nil, pl.WrapError(err, "failed to decode onion")
	}
	var o om.Onion
	if err = json.Unmarshal(onionBytes, &o); err != nil {
		return nil, pl.WrapError(err, "failed to unmarshal onion")
	}
	peeledSepal := o.Sepal.PeelSepal()
}

func BruiseOnion(onion string, privateKeyPEM string) {

}
