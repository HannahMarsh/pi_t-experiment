package keys

import (
	"reflect"
	"testing"
)

// TestEphemeralKeyGen tests the GenerateEphemeralKeyPair function.
func TestEphemeralKeyGen(t *testing.T) {
	relayPrivateKey, relayPublicKey, err := KeyGen()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ephemeralSharedSecret, ephemeralPublicKeyHex, err := GenerateEphemeralKeyPair(relayPublicKey)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that the shared secret is 32 bytes
	if reflect.TypeOf(ephemeralSharedSecret).Size() != 32 {
		t.Errorf("Expected shared secret to be 32 bytes, got %v bytes", reflect.TypeOf(ephemeralSharedSecret).Size())
	}

	// Generate shared secret with correct round
	computedSharedSecret, err := ComputeEphemeralSharedSecret(relayPrivateKey, ephemeralPublicKeyHex)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that the shared secret is 32 bytes
	if reflect.TypeOf(computedSharedSecret).Size() != 32 {
		t.Errorf("Expected shared secret to be 32 bytes, got %v bytes", reflect.TypeOf(computedSharedSecret).Size())
	}

	// Verify that the shared secret is the same
	if !reflect.DeepEqual(ephemeralSharedSecret, computedSharedSecret) {
		t.Errorf("Expected shared secret to be %v, got %v", ephemeralSharedSecret, computedSharedSecret)
	}

}
