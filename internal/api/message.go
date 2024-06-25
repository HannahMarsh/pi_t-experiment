package api

import (
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
)

type Message struct {
	From string
	To   string
	Msg  string
	Hash string
}

func NewMessage(from, to, msg string) Message {
	h := utils.GenerateUniqueHash()
	return Message{
		From: from,
		To:   to,
		Msg:  msg,
		Hash: h,
	}
}
