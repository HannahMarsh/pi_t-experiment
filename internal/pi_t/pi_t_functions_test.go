package pi_t

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestKeyGen(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	if privateKeyPEM == "" || publicKeyPEM == "" {
		t.Fatal("KeyGen() returned empty keys")
	}
}

func TestFormOnion(t *testing.T) {
	_, publicKeyPEM, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM, publicKeyPEM}
	routingPath := []string{"node1", "node2"}

	addr, onion, err := FormOnion(payload, publicKeys, routingPath)
	if err != nil {
		t.Fatalf("FormOnion() error: %v", err)
	}

	if addr != "node1" {
		t.Fatalf("FormOnion() expected address 'node1', got %s", addr)
	}

	if onion == "" {
		t.Fatal("FormOnion() returned empty onion")
	}
}

func TestPeelOnion(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM, publicKeyPEM}
	routingPath := []string{"node1", "node2"}

	_, onion, err := FormOnion(payload, publicKeys, routingPath)
	if err != nil {
		t.Fatalf("FormOnion() error: %v", err)
	}

	nextHop, peeledPayload, err := PeelOnion(onion, privateKeyPEM)
	if err != nil {
		t.Fatalf("PeelOnion() error: %v", err)
	}

	if nextHop != "node2" {
		t.Fatalf("PeelOnion() expected next hop 'node2', got %s", nextHop)
	}

	decodedPayload, err := base64.StdEncoding.DecodeString(peeledPayload)
	if err != nil {
		t.Fatalf("PeelOnion() error decoding payload: %v", err)
	}

	var layer OnionLayer
	err = json.Unmarshal(decodedPayload, &layer)
	if err != nil {
		t.Fatalf("PeelOnion() error unmarshaling layer: %v", err)
	}

	if layer.Payload != base64.StdEncoding.EncodeToString(payload) {
		t.Fatalf("PeelOnion() expected payload %s, got %s", base64.StdEncoding.EncodeToString(payload), layer.Payload)
	}
}

func TestBruiseOnion(t *testing.T) {
	payload := []byte("secret message")
	onion := base64.StdEncoding.EncodeToString(payload)

	bruisedOnion, err := BruiseOnion(onion)
	if err != nil {
		t.Fatalf("BruiseOnion() error: %v", err)
	}

	if bruisedOnion == onion {
		t.Fatal("BruiseOnion() did not modify the onion")
	}

	decodedBruisedOnion, err := base64.StdEncoding.DecodeString(bruisedOnion)
	if err != nil {
		t.Fatalf("BruiseOnion() error decoding bruised onion: %v", err)
	}

	if decodedBruisedOnion[0] != payload[0]^0xFF {
		t.Fatalf("BruiseOnion() did not correctly modify the onion, got %x", decodedBruisedOnion[0])
	}
}
