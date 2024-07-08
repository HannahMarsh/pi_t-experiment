package pi_t

import (
	_ "crypto/rand"
	"encoding/base64"
	_ "encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"hash/fnv"
	_ "strings"
)

const fixedLegnthOfMessage = 256

// FormOnion creates a forward onion from a message m, a path P, public keys pk, and metadata y.
// Parameters:
// - m: a fixed length message
// - P: a routing path (sequence of addresses representing l1 mixers and l2 gatekeepers such that len(P) = l1 + l2 + 1)
// - l1: the number of mixers in the routing path
// - l2: the number of gatekeepers in the routing path
// - pk: a list of public keys for the entities in the routing path
// - y: metadata associated with each entity (except the last destination entity) in the routing path
// Returns:
// - A list of lists of onions, O = (O_1, ..., O_l), where each O_i contains all possible variations of the i-th onion layer.
//   - The first list O_1 contains just the onion for the first mixer.
//   - For 2 <= i <= l1, the list O_i contains i options, O_i = (O_i,0, ..., O_i,i-1), each O_i,j representing the i-th onion layer with j prior bruises.
//   - For l1 + 1 <= i <= l1 + l2, the list O_i contains l1 + 1 options, depending on the total bruising from the mixers.
//   - The last list O_(l1 + l2 + 1) contains just the innermost onion for the recipient.
func FORMONION(publicKey, privateKey, m string, mixers []string, gatekeepers []string, recipient string, publicKeys []string, metadata []onion_model.Metadata, d int) ([][]onion_model.Onion, error) {

	message := padMessage(m)

	path := append(append(append([]string{""}, mixers...), gatekeepers...), recipient)
	l1 := len(mixers)
	l2 := len(gatekeepers)
	l := l1 + l2 + 1

	// Generate keys for each layer and the master key
	layerKeys := make([][]byte, l+1)
	for i := range layerKeys {
		layerKey, _ := keys.GenerateSymmetricKey()
		layerKeys[i] = layerKey //base64.StdEncoding.EncodeToString(layerKey)
	}
	K, _ := keys.GenerateSymmetricKey()
	masterKey := base64.StdEncoding.EncodeToString(K)

	// Construct first sepal for M1
	A, S, err := onion_model.FormSepals(masterKey, d, layerKeys, l, l1, l2, Hash)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create sepal")
	}

	// build penultimate onion layer

	// form content
	C, err := onion_model.FormContent(layerKeys, l, message, K)
	if err != nil {
		return nil, pl.WrapError(err, "failed to form content")
	}

	// form header
	H, err := onion_model.FormHeaders(l, l1, C, A, privateKey, publicKeys, recipient, layerKeys, K, path, Hash, metadata)

	// Initialize the onion structure
	onionLayers := make([][]onion_model.Onion, l+1)

	for i := range S {
		onionLayers[i] = utils.Map(S[i], func(sepal onion_model.Sepal) onion_model.Onion {
			return onion_model.Onion{
				Header:  H[i+1],
				Content: C[i+1],
				Sepal:   sepal,
			}
		})
	}

	return onionLayers, nil
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
