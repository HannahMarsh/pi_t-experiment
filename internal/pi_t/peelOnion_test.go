package pi_t

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"testing"
)

func TestPeelOnion22(t *testing.T) {
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

	onions, err := FORMONION(nodes[0].publicKeyPEM, nodes[0].privateKeyPEM, string(payload), routingPath[:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys, metadata, d)
	if err != nil {
		slog.Error("", err)
		t.Fatalf("failed")
	}

	for i, _ := range onions {
		for _, onion := range onions[i] {

			oBytes, err := json.Marshal(onion)
			if err != nil {
				slog.Error("failed to marshal onion", err)
				t.Fatalf("failed to marshal onion")
			}
			sharedKey, err := keys.ComputeSharedKey(nodes[i+1].privateKeyPEM, nodes[0].publicKeyPEM)
			if err != nil {
				slog.Error("failed to compute shared key", err)
				t.Fatalf("failed to compute shared key")
			}
			layer, metadata_, _, nextDestination, err := PeelOnion(base64.StdEncoding.EncodeToString(oBytes), sharedKey)

			if err != nil {
				slog.Error("failed to peel onion", err)
				t.Fatalf("failed to peel onion")
			}

			if layer != i+1 {
				t.Fatalf("layer does not match. Expected %d, got %d", i+1, layer)
			}

			if nextDestination != nodes[i+2].address {
				t.Fatalf("next destination does not match. Expected %s, got %s", nodes[i+2].address, nextDestination)
			}

			if metadata_.Example != metadata[i+1].Example {
				t.Fatalf("metadata does not match. Expected %s, got %s", metadata[i+1].Example, metadata_.Example)
			}
		}
	}

}
