package structs

import (
	"time"
)

type PublicNodeApi struct {
	ID        int
	Address   string
	PublicKey string
	IsMixer   bool
	Time      time.Time
}

type IntentToSend struct {
	From PublicNodeApi
	To   []PublicNodeApi
	Time time.Time
}
