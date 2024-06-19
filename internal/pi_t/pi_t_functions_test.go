package pi_t

import (
	"encoding/base64"
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
	privateKeyPEM1, publicKeyPEM1, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM2, publicKeyPEM2, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
	routingPath := []string{"node1", "node2"}

	destination, onion, err := FormOnion(payload, publicKeys, routingPath)
	if err != nil {
		t.Fatalf("FormOnion() error: %v", err)
	}

	if destination != "node1" {
		t.Fatalf("PeelOnion() expected destination to be 'node1', got %s", destination)
	}

	nextHop, peeledPayload, err := PeelOnion(onion, privateKeyPEM1)
	if err != nil {
		t.Fatalf("PeelOnion() error: %v", err)
	}

	if nextHop != "node2" {
		t.Fatalf("PeelOnion() expected next hop 'node1', got %s", nextHop)
	}

	nextHop2, peeledPayload2, err := PeelOnion(peeledPayload, privateKeyPEM2)
	if err != nil {
		t.Fatalf("PeelOnion() error: %v", err)
	}

	if nextHop2 != "" {
		t.Fatalf("PeelOnion() expected next hop '', got %s", nextHop2)
	}

	if peeledPayload2 != string(payload) {
		t.Fatalf("PeelOnion() expected payload %s, got %s", string(payload), peeledPayload2)
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
