package keys

import "testing"

func TestKeyGen(t *testing.T) {

	privateKeyPEM, publicKeyPEM, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	if privateKeyPEM == "" || publicKeyPEM == "" {
		t.Fatal("KeyGen() returned empty keys")
	}
}
