package api

import (
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
)

type Message struct {
	From string
	To   string
	Msg  string
	Hash string
}

func NewMessage(from, to, msg string) Message {
	h, err := utils.GenerateUniqueHash()
	if err != nil {
		slog.Error("failed to generate unique hash", err)
		h = ""
	}
	return Message{
		From: from,
		To:   to,
		Msg:  msg,
		Hash: h,
	}
}
