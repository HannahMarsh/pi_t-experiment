package pi_t

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/keys"
	"testing"
)

func TestFormOnion(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM, publicKeyPEM}
	routingPath := []string{"node1", "node2"}

	addr, onion, err := FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1)
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
	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	// client processing

	payload := []byte("secret message")
	publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
	routingPath := []string{"node1", "node2"}

	destination, onion, err := FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1)
	if err != nil {
		t.Fatalf("FormOnion() error: %v", err)
	}

	if destination != "node1" {
		t.Fatalf("PeelOnion() expected destination to be 'node1', got %s", destination)
	}

	// first hop processing

	peeled, bruises, _, err := PeelOnion(onion, privateKeyPEM1)
	if err != nil {
		t.Fatalf("PeelOnion() error: %v", err)
	}

	if bruises != 0 {
		t.Fatalf("PeelOnion() expected bruises 0, got %d", bruises)
	}
	if peeled.NextHop != "node2" {
		t.Fatalf("PeelOnion() expected next hop 'node1', got %s", peeled.NextHop)
	}

	headerAdded, err := AddHeader(peeled, 1, privateKeyPEM1, publicKeyPEM1)

	// second hop processing

	peeled2, bruises2, _, err := PeelOnion(headerAdded, privateKeyPEM2)
	if err != nil {
		t.Fatalf("PeelOnion() error: %v", err)
	}
	if bruises2 != 1 {
		t.Fatalf("PeelOnion() expected bruises 1, got %d", bruises2)
	}

	if peeled2.NextHop != "" {
		t.Fatalf("PeelOnion() expected next hop '', got %s", peeled2.NextHop)
	}

	if peeled2.Payload != string(payload) {
		t.Fatalf("PeelOnion() expected payload %s, got %s", string(payload), peeled.Payload)
	}
}

func TestNonceVerification(t *testing.T) {
	for i := 0; i < 100; i++ {
		privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}
		privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
		if err != nil {
			t.Fatalf("KeyGen() error: %v", err)
		}

		// Client processing
		payload := []byte("secret message")
		publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
		routingPath := []string{"node1", "node2"}

		destination, onion, err := FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1)
		if err != nil {
			t.Fatalf("FormOnion() error: %v", err)
		}

		if destination != "node1" {
			t.Fatalf("PeelOnion() expected destination to be 'node1', got %s", destination)
		}

		// First hop processing with nonce verification
		peeled, bruises, nonceVerification, err := PeelOnion(onion, privateKeyPEM1)
		if err != nil {
			t.Fatalf("PeelOnion() error: %v", err)
		}

		if bruises != 0 {
			t.Fatalf("PeelOnion() expected bruises 0, got %d", bruises)
		}
		if peeled.NextHop != "node2" {
			t.Fatalf("PeelOnion() expected next hop 'node2', got %s", peeled.NextHop)
		}

		// Check nonce verification
		if !nonceVerification {
			t.Fatalf("PeelOnion() nonce verification failed")
		}

		headerAdded, err := AddHeader(peeled, 1, privateKeyPEM1, publicKeyPEM1)
		if err != nil {
			t.Fatalf("AddHeader() error: %v", err)
		}

		// Second hop processing with nonce verification
		peeled2, bruises2, nonceVerification2, err := PeelOnion(headerAdded, privateKeyPEM2)
		if err != nil {
			t.Fatalf("PeelOnion() error: %v", err)
		}
		if bruises2 != 1 {
			t.Fatalf("PeelOnion() expected bruises 1, got %d", bruises2)
		}

		if peeled2.NextHop != "" {
			t.Fatalf("PeelOnion() expected next hop '', got %s", peeled2.NextHop)
		}

		if peeled2.Payload != string(payload) {
			t.Fatalf("PeelOnion() expected payload %s, got %s", string(payload), peeled2.Payload)
		}

		// Check nonce verification for second hop
		if !nonceVerification2 {
			t.Fatalf("PeelOnion() nonce verification failed on second hop")
		}
	}
}
