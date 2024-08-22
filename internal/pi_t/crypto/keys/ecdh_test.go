package keys

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

func TestEncryptStringWithAES(t *testing.T) {
	priv1, pub1, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}
	priv2, pub2, err := KeyGen()
	if err != nil {
		t.Fatalf("KeyGen() error: %v", err)
	}

	sharedKey1, err := ComputeSharedKey(priv1, pub2)
	if err != nil {
		t.Fatalf("ComputeSharedKey() error: %v", err)
	}

	sharedKey2, err := ComputeSharedKey(priv2, pub1)
	if err != nil {
		t.Fatalf("ComputeSharedKey() error: %v", err)
	}

	if sharedKey1 != sharedKey2 {
		t.Fatalf("shared keys are different")
	}

	type M struct {
		Msg string
	}

	var m M = M{"hello"}

	mBytes, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	mString := base64.StdEncoding.EncodeToString(mBytes)

	_, encrypted, err := EncryptStringWithAES(sharedKey1[:], mString)
	if err != nil {
		t.Fatalf("EncryptWithAES() error: %v", err)
	}

	decrypted, _, err := DecryptStringWithAES(sharedKey2[:], encrypted)
	if err != nil {
		t.Fatalf("DecryptStringWithAES() error: %v", err)
	}

	var m2 M
	err = json.Unmarshal(decrypted, &m2)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if m.Msg != m2.Msg {
		t.Fatalf("expected %v, got %v", m.Msg, m2.Msg)
	}

}
