package keys

import (
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"reflect"
	"testing"
)

// TestEphemeralKeyGen tests the GenerateEphemeralKeyPair function.
func TestEphemeralKeyGen(t *testing.T) {
	relayPrivateKey, relayPublicKey, err := KeyGen()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	rounds := utils.NewIntArray(1, 100)
	for _, round := range rounds {

		ephemeralSharedSecret, ephemeralPublicKeyHex, err := GenerateEphemeralKeyPair(round, relayPublicKey)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify that the shared secret is 32 bytes
		if reflect.TypeOf(ephemeralSharedSecret).Size() != 32 {
			t.Errorf("Expected shared secret to be 32 bytes, got %v bytes", reflect.TypeOf(ephemeralSharedSecret).Size())
		}

		// Generate shared secret with correct round
		computedSharedSecret, err := ComputeEphemeralSharedSecret(round, relayPrivateKey, ephemeralPublicKeyHex)
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

		for _, incorrectRound := range rounds {
			if incorrectRound == round {
				continue
			}

			// Generate shared secret with incorrect round
			badKey, err := ComputeEphemeralSharedSecret(incorrectRound, relayPrivateKey, ephemeralPublicKeyHex)
			if err == nil {
				t.Errorf("Expected error, got nil")
			} else {
				// Verify that the shared secret is not the same
				if reflect.DeepEqual(ephemeralSharedSecret, badKey) {
					t.Errorf("Expected keys to not be the same")
				}
			}
		}
	}
}
