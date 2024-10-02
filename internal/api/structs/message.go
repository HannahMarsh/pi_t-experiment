package structs

import (
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
)

type Message struct {
	From string `json:"f"`
	To   string `json:"t"`
	Msg  string `json:"msg"`
	Hash string `json:"h"`
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
