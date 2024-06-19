package api

import "time"

type PublicNodeApi struct {
	ID        int
	Address   string
	PublicKey string
}

type PrivateNodeApi struct {
	TimeOfRequest time.Time
	ID            int
	Address       string
	PublicKey     string
	MessageQueue  []int
}
