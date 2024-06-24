package api

import (
	"crypto/ecdh"
	"time"
)

type PublicClientApi struct {
	ID        int
	Address   string
	PublicKey *ecdh.PublicKey
}

type IntentToSend struct {
	From PublicClientApi
	To   []PublicClientApi
	Time time.Time
}
