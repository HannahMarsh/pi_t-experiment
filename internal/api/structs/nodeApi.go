package structs

import (
	"time"
)

type PublicNodeApi struct {
	ID             int       `json:"i"`
	Address        string    `json:"a"`
	Host           string    `json:"h"`
	Port           int       `json:"po"`
	PrometheusPort int       `json:"pp"`
	PublicKey      string    `json:"pk"`
	Time           time.Time `json:"t"`
}

type IntentToSend struct {
	From PublicNodeApi   `json:"f"`
	To   []PublicNodeApi `json:"t"`
	Time time.Time       `json:"ti"`
}
