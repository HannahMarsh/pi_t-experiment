package api

import "time"

type PublicNodeApi struct {
	ID        int
	Address   string
	PublicKey []byte
}

type PrivateNodeApi struct {
	TimeOfRequest time.Time
	ID            int
	Address       string
	PublicKey     []byte
	MessageQueue  []int
}
