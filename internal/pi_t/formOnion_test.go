package pi_t

import (
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"strings"
	"testing"
)

func TestFORMONION(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	var err error

	l1 := 5
	l2 := 5
	d := 3
	l := l1 + l2 + 1

	type node struct {
		privateKeyPEM string
		publicKeyPEM  string
		address       string
	}

	nodes := make([]node, l+1)

	for i := 0; i < l+1; i++ {
		privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		nodes[i] = node{privateKeyPEM, publicKeyPEM, fmt.Sprintf("node%d", i)}
	}

	secretMessage := "secret message"

	payload, err := json.Marshal(structs.Message{
		Msg:  secretMessage,
		To:   nodes[l].address,
		From: nodes[0].address,
	})
	if err != nil {
		slog.Error("json.Marshal() error", err)
		t.Fatalf("json.Marshal() error: %v", err)
	}

	publicKeys := utils.Map(nodes[1:], func(n node) string { return n.publicKeyPEM })
	routingPath := utils.Map(nodes[1:], func(n node) string { return n.address })

	metadata := make([]onion_model.Metadata, l+1)
	for i := 0; i < l+1; i++ {
		metadata[i] = onion_model.Metadata{Example: fmt.Sprintf("example%d", i)}
	}

	onions, err := FORMONION(nodes[0].publicKeyPEM, nodes[0].privateKeyPEM, string(payload), routingPath[1:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys[1:], metadata, d)
	if err != nil {
		slog.Error("", err)
		t.Fatalf("failed")
	}
	h := Hash(strings.Join(onions[0][0].Sepal.Blocks, ""))
	if onions[0][0].Header.A[0] != h {
		t.Fatalf("Expected Hash to match")
	}

	for i, layer := range onions {
		for j, onion := range layer {
			h := Hash(strings.Join(onion.Sepal.Blocks, ""))
			if !utils.Contains(onion.Header.A, func(str string) bool {
				return str == h
			}) {
				t.Fatalf("Expected Hash to match i=%d, j = %d", i, j)
			}
		}
	}

}
