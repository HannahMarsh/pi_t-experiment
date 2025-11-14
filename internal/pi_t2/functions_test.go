package pi_t2

import (
	"crypto"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	pl "github.com/HannahMarsh/PrettyLogger"
)

// makeTestPath builds a simple OnionPath with NextHop hops and threshold t.
func makeTestPath(t *testing.T, numMixers int, numGatekeepers int) []Hop {

	n := numMixers + numGatekeepers + 1
	curve := ecdh.X25519()

	pks := make([]crypto.PublicKey, n)
	sks := make([]crypto.PrivateKey, n)
	addrs := make([]string, n)
	for i := 0; i < n; i++ {
		key, err := curve.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		pks[i] = key.PublicKey()
		sks[i] = key
		addrs[i] = fmt.Sprintf("hop %d", i)
	}

	hops, err := MakePath(pks, addrs, numMixers, numGatekeepers)
	if err != nil {
		t.Fatalf("failed to make path: %v", err)
	}

	for i := 0; i < n; i++ {
		hops[i].PrivateKey = sks[i]
	}
	return hops
}

func TestFormAndPeelOnion_NoBruise(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")

	ell_1 := 3 // num mixers
	ell_2 := 2 // num gatekeepers
	n := ell_1 + ell_2 + 1
	d := 1

	hops := makeTestPath(t, ell_1, ell_2)
	//msg := "hello anonymous world"

	msg_size_bytes := 256

	msg_bytes := make([]byte, msg_size_bytes)
	for i := range msg_bytes {
		msg_bytes[i] = 0xFF
	}
	copy(msg_bytes, []byte("hello anonymous world"))

	msg := string(msg_bytes)

	onion, err := formOnion([]byte(msg), hops, d)
	if err != nil {
		slog.Error("form Onion failed", "err", err)
		t.Fatalf("FormOnion failed: %v", err)
	}

	onion_bytes, err := json.Marshal(onion)
	if err != nil {
		slog.Error("marshal onion failed", "err", err)
	}
	size := len(onion_bytes)
	slog.Info("onion size", "size (KB)", int(float64(size)/1024.0))

	var peeled *PeeledOnion = nil
	current := onion

	// peel as mixers
	for i := 0; i < ell_1; i++ {
		hop := hops[i]
		peeled, err = PeelOnion(current, hop.PrivateKey)
		if err != nil {
			slog.Error("peel Onion failed", "err", err)
			t.Fatalf("PeelOnion failed at mixer, hop %d: %v", i, err)
		}
		nextHop := hops[i+1].Address
		if peeled.NextHop != nextHop {
			t.Fatalf("expected next hop to be %s, got %s", nextHop, peeled.NextHop)
		}
		if peeled.Role != MIXER {
			t.Fatalf("expected peeled Role to be %s, got %s", MIXER, peeled.Role)
		}
		current = peeled.NextOnion
	}

	// peel as gatekeeper
	for i := ell_1; i < n-2; i++ {
		hop := hops[i]
		peeled, err = PeelOnion(current, hop.PrivateKey)
		if err != nil {
			t.Fatalf("PeelOnion failed at gatekeeper, hop %d: %v", i, err)
		}
		nextHop := hops[i+1].Address
		if peeled.NextHop != nextHop {
			t.Fatalf("expected next hop to be %s, got %s", nextHop, peeled.NextHop)
		}
		if peeled.Role != GATEKEEPER {
			t.Fatalf("expected peeled Role to be %s, got %s", GATEKEEPER, peeled.Role)
		}
		current = peeled.NextOnion
	}

	// last gatekeeper
	hop := hops[n-2]
	peeled, err = PeelOnion(current, hop.PrivateKey)
	if err != nil {
		t.Fatalf("PeelOnion failed at last gatekeeper, hop %d: %v", n-2, err)
	}
	nextHop := hops[n-1].Address
	if peeled.NextHop != nextHop {
		t.Fatalf("expected next hop to be %s, got %s", nextHop, peeled.NextHop)
	}
	if peeled.Role != LAST_GATEKEEPER {
		t.Fatalf("expected peeled Role to be %s, got %s", LAST_GATEKEEPER, peeled.Role)
	}
	current = peeled.NextOnion

	// final recipient
	hop = hops[n-1]
	peeled, err = PeelOnion(current, hop.PrivateKey)
	if err != nil {
		t.Fatalf("PeelOnion failed at final recipient, hop %d: %v", n-1, err)
	}
	if peeled.Role != RECIPIENT {
		t.Fatalf("expected peeled Role to be %s, got %s", RECIPIENT, peeled.Role)
	}
	peeled_msg := string(peeled.Content)
	if peeled_msg != msg {
		t.Fatalf("expected peeled message to be \"%s\", got \"%s\"", msg, peeled_msg)
	}
}

//
//func TestSepalLengthAfterPeel(t *testing.T) {
//	path := makeTestPath(t, 2, 1)
//	msg := []byte("test message")
//
//	onion, _ := FormOnion(msg, path)
//	initialSepal := len(onion.Sepal)
//
//	next, _ := PeelOnion(onion, path.Hops[0].PrivateKey)
//	if len(next.Sepal) != initialSepal-1 {
//		t.Fatalf("expected sepal length %d after peel, got %d", initialSepal-1, len(next.Sepal))
//	}
//}
//
//func TestBruiseOnion_CountsCorrectly(t *testing.T) {
//	path := makeTestPath(t, 2, 1)
//	msg := []byte("bruisable")
//
//	onion, _ := FormOnion(msg, path)
//	next, err := BruiseOnion(onion, path.Hops[0].PrivateKey)
//	if err != nil {
//		t.Fatalf("BruiseOnion failed: %v", err)
//	}
//
//	if next.Bruises != 1 {
//		t.Fatalf("expected 1 bruise, got %d", next.Bruises)
//	}
//
//	if len(next.Sepal) != len(onion.Sepal)-1 {
//		t.Fatalf("expected sepal length to shrink after bruising")
//	}
//}
//
//func TestExceedingThresholdFails(t *testing.T) {
//	// Threshold = 1, 2 hops. Bruise both times → final onion should not decrypt properly
//	path := makeTestPath(t, 2, 1)
//	msg := []byte("dead message")
//
//	onion, _ := FormOnion(msg, path)
//
//	// Bruise at hop 0
//	o1, err := BruiseOnion(onion, path.Hops[0].PrivateKey)
//	if err != nil {
//		t.Fatalf("BruiseOnion failed at hop 0: %v", err)
//	}
//
//	// Bruise at hop 1 (beyond threshold)
//	o2, err := BruiseOnion(o1, path.Hops[1].PrivateKey)
//	if err != nil {
//		t.Fatalf("BruiseOnion failed at hop 1: %v", err)
//	}
//
//	// Try to decrypt final — since sepal fully bruised, content should NOT match original
//	if o2.Content != nil && string(o2.Content) == string(msg) {
//		t.Fatalf("expected final onion to fail after exceeding threshold")
//	}
//
//}
//
//func TestMixedPeelAndBruise(t *testing.T) {
//	path := makeTestPath(t, 3, 2)
//	msg := []byte("mixed strategy")
//
//	onion, _ := FormOnion(msg, path)
//	o1, _ := PeelOnion(onion, path.Hops[0].PrivateKey) // peel hop 0
//	o2, _ := BruiseOnion(o1, path.Hops[1].PrivateKey)  // bruise hop 1
//	o3, _ := PeelOnion(o2, path.Hops[2].PrivateKey)    // peel hop 2
//
//	// Check bruises count propagated
//	if o3.Bruises != 1 {
//		t.Fatalf("expected 1 bruise, got %d", o3.Bruises)
//	}
//
//	// Since threshold = 2, message should still decrypt correctly
//	if string(o3.Content) != string(msg) {
//		t.Fatalf("expected decrypted message %q, got %q", string(msg), string(o3.Content))
//	}
//}
