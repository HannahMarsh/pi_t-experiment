package onion_model

import (
	"encoding/base64"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"strings"
	"testing"
)

//func Hash(s string) string {
//	h := fnv.New32a()
//	_, err := h.Write([]byte(s))
//	if err != nil {
//		slog.Error("failed to Hash string", err)
//		return ""
//	}
//	return fmt.Sprint(h.Sum32())
//}

func TestFormSepal(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")
	l1 := 5
	l2 := 5
	d := 3
	l := l1 + l2 + 1

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
	_, sepals, err := FormSepals(masterKey, d, layerKeys, l, l1, l2, Hash)
	if err != nil {
		slog.Error("failed to create sepal", err)
		t.Fatalf("formSepals() error: %v", err)
	}

	sepal := sepals[0][0]

	if len(sepal.Blocks) != l1+1 {
		t.Fatalf("formSepals() expected %d blocks, got %d", l1+1, len(sepal.Blocks))
	}

	for j, sepalBlock := range sepal.Blocks {

		if j < d {
			decryptedBlock, _, err := keys.DecryptStringWithAES(layerKeys[1], sepalBlock)
			if err != nil {
				slog.Error("failed to decrypt sepal block", err)
				t.Fatalf("DecryptStringWithAES() error: %v", err)
			}
			for i := 2; i <= l-1; i++ {
				k := layerKeys[i]

				decryptedBlock, _, err = keys.DecryptWithAES(k, decryptedBlock)
				if err != nil {
					slog.Error("failed to decrypt sepal block", err)
					t.Fatalf("DecryptStringWithAES() error: %v", err)
				}
			}
			for index, b := range K {
				keyblockByte := decryptedBlock[index]
				if keyblockByte != b {
					slog.Info("failed to decrypt sepal block. Expected master key")
					t.Fatalf("formSepals() expected keyblock %v, got %v", b, keyblockByte)
				}
			}
		} else {
			decryptedBlock, _, err := keys.DecryptStringWithAES(layerKeys[1], sepalBlock)
			if err != nil {
				slog.Error("failed to decrypt sepal block", err)
				t.Fatalf("DecryptStringWithAES() error: %v", err)
			}
			var decryptedString string
			for i := 2; i <= l1+1; i++ {
				k := layerKeys[i]

				decryptedBlock, decryptedString, err = keys.DecryptWithAES(k, decryptedBlock)
				if err != nil {
					slog.Error("failed to decrypt sepal block", err)
					t.Fatalf("DecryptStringWithAES() error: %v", err)
				}
			}
			if !strings.HasPrefix(decryptedString, "null") {
				t.Fatalf("formSepals() expected decryptedString to start with 'null', got %v", decryptedString)
			}
			//if len(decryptedString) != (saltLength * 8) {
			//	t.Fatalf("formSepals() expected decryptedString length %v, got %v", (saltLength * 8), len(decryptedString))
			//}
		}
	}
}

func TestBruiseSepal(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")
	l1 := 5
	l2 := 5
	d := 3
	l := l1 + l2 + 1

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
	_, sepals, err := FormSepals(masterKey, d, layerKeys, l, l1, l2, Hash)
	if err != nil {
		slog.Error("failed to create sepal", err)
		t.Fatalf("formSepals() error: %v", err)
	}
	sepal := sepals[0][0]

	if len(sepal.Blocks) != l1+1 {
		t.Fatalf("formSepals() expected %d blocks, got %d", l1+1, len(sepal.Blocks))
	}

	bruised, err := bruiseSepal(sepal, layerKeys, d-1, l1, l, d)
	if err != nil {
		slog.Error("failed to bruise sepal", err)
		t.Fatalf("bruiseSepal() error: %v", err)
	}

	if len(bruised.Blocks) != 1 {
		t.Fatalf("bruiseSepal() expected 1 block, got %d", len(bruised.Blocks))
	}

	block := bruised.Blocks[0]

	for i := l1 + 1; i <= l-1; i++ {
		_, block, err = keys.DecryptStringWithAES(layerKeys[i], block)
		if err != nil {
			slog.Error("failed to decrypt sepal block", err)
			t.Fatalf("DecryptStringWithAES() error: %v", err)
		}
		slog.Info("decrypted block: ", "block", block)
	}

	slog.Info("bruised sepal block: ", "block", block)

	keyblockBytes, err := base64.StdEncoding.DecodeString(block)
	if err != nil {
		slog.Error("failed to decode sepal block", err)
		t.Fatalf("DecodeString() error: %v", err)
	}

	for index, b := range K {
		keyblockByte := keyblockBytes[index]
		if keyblockByte != b {
			slog.Info("failed to decrypt sepal block. Expected master key")
			t.Fatalf("formSepals() expected keyblock %v, got %v", b, keyblockByte)
		}
	}

	// bruise it d times
	bruised, err = bruiseSepal(sepal, layerKeys, d, l1, l, d)
	if err != nil {
		slog.Error("failed to bruise sepal", err)
		t.Fatalf("bruiseSepal() error: %v", err)
	}

	if len(bruised.Blocks) != 1 {
		t.Fatalf("bruiseSepal() expected 1 block, got %d", len(bruised.Blocks))
	}

	block = bruised.Blocks[0]

	_, block, err = keys.DecryptStringWithAES(layerKeys[l1+1], block)
	if err != nil {
		slog.Error("failed to decrypt sepal block", err)
		t.Fatalf("DecryptStringWithAES() error: %v", err)
	}
	slog.Info("decrypted block: ", "block", block)

	if !strings.HasPrefix(block, "null") {
		t.Fatalf("formSepals() expected decryptedString to start with 'null', got %v", bruised.Blocks[0])
	}
}

func bruiseSepal(sepal Sepal, layerKeys [][]byte, numBruises int, l1 int, l int, d int) (s Sepal, err error) {
	randomBools := make([]bool, l1)
	for i := range randomBools {
		if i < numBruises {
			randomBools[i] = true
		} else {
			randomBools[i] = false
		}
	}
	utils.Shuffle(randomBools)
	randomBools = append([]bool{false}, randomBools...)

	for i := 1; i <= l1; i++ {
		dobruiseSepal := false
		if i <= l1 {
			dobruiseSepal = randomBools[i]
		}
		sepal, err = sepal.PeelSepal(layerKeys[i])
		if dobruiseSepal {
			sepal.AddBruise()
		} else {
			sepal.RemoveBlock()
		}
		if err != nil {
			return Sepal{}, pl.WrapError(err, "failed to peel sepal")
		}
	}
	return sepal, nil
}

func TestSepalHashes(t *testing.T) {
	pl.SetUpLogrusAndSlog("debug")
	l1 := 5
	l2 := 5
	d := 3
	l := l1 + l2 + 1

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
	A, sepals, err := FormSepals(masterKey, d, layerKeys, l, l1, l2, Hash)
	if err != nil {
		slog.Error("failed to create sepal", err)
		t.Fatalf("formSepals() error: %v", err)
	}

	sepal := sepals[0][0]

	h := Hash(strings.Join(sepal.Blocks, ""))
	if A[0][0] != h {
		t.Fatalf("hash not expected")
	}

	for i := 1; i <= l1; i++ {

		sepal, err = sepal.PeelSepal(layerKeys[i])
		if err != nil {
			slog.Error("failed to peel sepal", err)
			t.Fatalf("PeelSepal() error: %v", err)
		}
		sepal.RemoveBlock()
		h = Hash(strings.Join(sepal.Blocks, ""))
		if !utils.Contains(A[i], func(str string) bool {
			return str == h
		}) {
			t.Fatalf("hash not expected")
		}
	}

}
