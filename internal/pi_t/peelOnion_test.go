package pi_t

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	om "github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"testing"
)

func TestPeelOnion(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	var err error

	l1 := 3
	l2 := 2
	d := 2
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

	metadata := make([]om.Metadata, l+1)
	for i := 0; i < l+1; i++ {
		metadata[i] = om.Metadata{Nonce: ""}
	}

	onions, err := FORMONION(nodes[0].privateKeyPEM, string(payload), routingPath[:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys, metadata, d)
	if err != nil {
		slog.Error("", err)
		t.Fatalf("failed")
	}

	o := onions[0][0]

	for i := 1; i <= l1+l2+1; i++ {
		data, err := json.Marshal(o)
		if err != nil {
			slog.Error("failed to marshal onion", err)
			t.Fatalf("failed to marshal onion")
		}

		oStr := base64.StdEncoding.EncodeToString(data)
		mixer := nodes[i]
		role, layer, _, peeled, nextHop, err := PeelOnion(oStr, mixer.privateKeyPEM)

		if role == om.MIXER {
			peeled.Sepal = peeled.Sepal.RemoveBlock()
		}

		if i <= l1 {
			if role != om.MIXER {
				t.Fatalf("role does not match. Expected %s, got %s", om.MIXER, role)
			}
			//peeled.Sepal = peeled.Sepal.RemoveBlock()
		}
		if i > l1 && i < l1+l2 && role != om.GATEKEEPER {
			t.Fatalf("role does not match. Expected %s, got %s", om.GATEKEEPER, role)
		}
		if i == l1+l2 && role != om.LAST_GATEKEEPER {
			t.Fatalf("role does not match. Expected %s, got %s", om.LAST_GATEKEEPER, role)
		}

		if layer != i {
			t.Fatalf("layer does not match. Expected %d, got %d", i, layer)
		}

		if i <= l1+l2 && nextHop != nodes[i+1].address {
			t.Fatalf("next hop does not match. Expected %s, got %s", nodes[i+1].address, nextHop)
		}

		if i == l1+l2+1 {
			var receivedMessage structs.Message
			err := json.Unmarshal([]byte(peeled.Content), &receivedMessage)
			if err != nil {
				slog.Error("failed to unmarshal message", err)
				t.Fatalf("failed to unmarshal message")
			}

			if receivedMessage.Msg != secretMessage {
				t.Fatalf("message does not match. Expected %s, got %s", secretMessage, receivedMessage.Msg)
			}
			if receivedMessage.From != nodes[0].address {
				t.Fatalf("from does not match. Expected %s, got %s", nodes[0].address, receivedMessage.From)
			}
			if receivedMessage.To != nodes[l].address {
				t.Fatalf("to does not match. Expected %s, got %s", nodes[l].address, receivedMessage.To)
			}
		}

		o = peeled

	}
}

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

	metadata := make([]om.Metadata, l+1)
	for i := 0; i < l+1; i++ {
		metadata[i] = om.Metadata{Example: fmt.Sprintf("example%d", i)}
	}

	onions, err := FORMONION(nodes[0].privateKeyPEM, string(payload), routingPath[:l1], routingPath[l1:len(routingPath)-1], routingPath[len(routingPath)-1], publicKeys, metadata, d)
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
			role, layer, metadata_, peeled, nextDestination, err := PeelOnion(base64.StdEncoding.EncodeToString(oBytes), nodes[i+1].privateKeyPEM)

			if err != nil {
				slog.Error("failed to peel onion", err)
				t.Fatalf("failed to peel onion")
			}

			if layer != i+1 {
				t.Fatalf("layer does not match. Expected %d, got %d", i+1, layer)
			}
			if i+2 < len(nodes) {
				if nextDestination != nodes[i+2].address {
					t.Fatalf("next destination does not match. Expected %s, got %s", nodes[i+2].address, nextDestination)
				}
			}

			if metadata_.Example != metadata[i+1].Example {
				t.Fatalf("metadata does not match. Expected %s, got %s", metadata[i+1].Example, metadata_.Example)
			}

			if i < l1 {
				if role != om.MIXER {
					t.Fatalf("role does not match. Expected %s, got %s", om.MIXER, role)
				}
			} else if i < l-2 && role != om.GATEKEEPER {
				t.Fatalf("role does not match. Expected %s, got %s", om.GATEKEEPER, role)
			}
			if i == l-2 && role != om.LAST_GATEKEEPER {
				t.Fatalf("role does not match. Expected %s, got %s", om.LAST_GATEKEEPER, role)
			}

			if i == l-1 {
				var receivedMessage structs.Message
				err := json.Unmarshal([]byte(peeled.Content), &receivedMessage)
				if err != nil {
					slog.Error("failed to unmarshal message", err)
					t.Fatalf("failed to unmarshal message")
				}

				if receivedMessage.Msg != secretMessage {
					t.Fatalf("message does not match. Expected %s, got %s", secretMessage, receivedMessage.Msg)
				}
				if receivedMessage.From != nodes[0].address {
					t.Fatalf("from does not match. Expected %s, got %s", nodes[0].address, receivedMessage.From)
				}
				if receivedMessage.To != nodes[l].address {
					t.Fatalf("to does not match. Expected %s, got %s", nodes[l].address, receivedMessage.To)
				}
			}
		}
	}

}
