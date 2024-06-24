package api

import (
	"crypto/ecdh"
	"time"
)

type PublicNodeApi struct {
	ID        int
	Address   string
	PublicKey *ecdh.PublicKey
}

type PrivateNodeApi struct {
	TimeOfRequest time.Time
	ID            int
	Address       string
	PublicKey     *ecdh.PublicKey
	MessageQueue  []int
}
