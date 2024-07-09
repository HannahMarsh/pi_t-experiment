package onion_model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"hash/fnv"
	"strings"
	"testing"
)

func TestFormHeader(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

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

	message := padMessage(base64.StdEncoding.EncodeToString(payload))
	// Generate keys for each layer and the master key
	layerKeys := make([][]byte, l+1)
	for i := range layerKeys {
		layerKey, _ := keys.GenerateSymmetricKey()
		layerKeys[i] = layerKey //base64.StdEncoding.EncodeToString(layerKey)
	}
	K, err := keys.GenerateSymmetricKey()
	if err != nil {
		slog.Error("failed to generate symmetric key", err)
		t.Fatalf("GenerateSymmetricKey() error: %v", err)
	}
	masterKey := base64.StdEncoding.EncodeToString(K)

	// Construct first sepal for M1
	A, _, err := FormSepals(masterKey, d, layerKeys, l, l1, l2, Hash)
	if err != nil {
		t.Fatalf("failed to create sepal")
	}

	// build penultimate onion layer

	// form content
	C, err := FormContent(layerKeys, l, message, K)
	if err != nil {
		t.Fatalf("failed to form content")
	}

	recipient := routingPath[len(routingPath)-1]

	metadata := make([]Metadata, l+1)
	for i := 0; i < l+1; i++ {
		metadata[i] = Metadata{Example: fmt.Sprintf("example%d", i)}
	}

	// form header
	H, err := FormHeaders(l, l1, C, A, nodes[0].privateKeyPEM, publicKeys, recipient, layerKeys, K, append([]string{""}, routingPath...), Hash, metadata)

	for i, h := range H[1:] {
		sharedKey, err := keys.ComputeSharedKey(nodes[i+1].privateKeyPEM, nodes[0].publicKeyPEM)
		if err != nil {
			slog.Error("failed to compute shared key", err)
			t.Fatalf("failed to compute shared key")
		}
		cypherText, nextHop, nextHeader, err := h.DecodeHeader(sharedKey)
		if err != nil {
			slog.Error("failed to decode header", err)
			t.Fatalf("failed to decode header")
		}
		if i < len(H)-2 {
			if nextHeader.NextHeader != H[i+2].NextHeader {
				t.Fatalf("Expected next header to match")
			}
			if nextHeader.E != H[i+2].E {
				t.Fatalf("Expected E to match")
			}
			if strings.Join(nextHeader.A, "") != strings.Join(H[i+2].A, "") {
				t.Fatalf("Expected A to match")
			}
		}
		//slog.Info("", "", nextHeader)

		if i < l1 && cypherText.Recipient != "mixer" {
			t.Fatalf("Expected mixer")
		} else if i >= l1 && i < l1+l2-1 && cypherText.Recipient != "gatekeeper" {
			t.Fatalf("Expected gatekeeper")
		} else if i == l1+l2-1 && cypherText.Recipient != "lastGatekeeper" {
			t.Fatalf("Expected lastGatekeeper")
		} else if i == l1+l2 && cypherText.Recipient != recipient {
			t.Fatalf("Expected recipient")
		}
		if cypherText.Layer != i+1 {
			t.Fatalf("Expected layer to match")
		}
		if i != l1+l2 && nextHop != routingPath[i+1] {
			t.Fatalf("Expected address to match")
		}
	}
}

func Hash(s string) string {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		slog.Error("failed to Hash string", err)
		return ""
	}
	return fmt.Sprint(h.Sum32())
}

func padMessage(message string) []byte {
	var nullTerminator byte = '\000'
	var paddedMessage = make([]byte, fixedLegnthOfMessage)
	var mLength = len(message)

	for i := 0; i < fixedLegnthOfMessage; i++ {
		if i >= mLength || i == fixedLegnthOfMessage-1 {
			paddedMessage[i] = nullTerminator
		} else {
			paddedMessage[i] = message[i]
		}
	}
	return paddedMessage
}

const fixedLegnthOfMessage = 256
