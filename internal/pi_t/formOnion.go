package pi_t

import (
	_ "crypto/rand"
	"encoding/base64"
	_ "encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"hash/fnv"
	"log/slog"
	_ "strings"
)

const fixedLengthOfMessage = 256

// FORMONION creates a forward onion from a message, path, public keys, and metadata.
//
// Parameters:
// - m: a fixed length message
// - recipient: the intended client receiver
// - publicKeys: a list of public keys for the entities in the routing path
// - metadata: metadata associated with each entity (except the last destination entity) in the routing path
// Returns:
// - A list of lists of onions, O = (O_1, ..., O_l), where each O_i contains all possible variations of the i-th onion layer.
//   - The first list O_1 contains just the onion for the first mixer.
//   - For 2 <= i <= l1, the list O_i contains i options, O_i = (O_i,0, ..., O_i,i-1), each O_i,j representing the i-th onion layer with j prior bruises.
//   - For l1 + 1 <= i <= l1 + l2, the list O_i contains l1 + 1 options, depending on the total bruising from the mixers.
//   - The last list O_(l1 + l2 + 1) contains just the innermost onion for the recipient.
func FORMONION(m string, mixers, gatekeepers []string, recipient string, publicKeys []string, metadata []onion_model.Metadata, d int) ([][]onion_model.Onion, error) {

	// Pad the message to a fixed length.
	message := padMessage(m)

	// Construct the full path for the onion, including mixers, gatekeepers, and the recipient.
	path := append(append(append([]string{""}, mixers...), gatekeepers...), recipient)
	l1 := len(mixers)      // Number of mixers.
	l2 := len(gatekeepers) // Number of gatekeepers.
	l := l1 + l2 + 1       // Total number of layers (mixers + gatekeepers + recipient).

	// Generate symmetric keys for each layer and the master key.
	layerKeys := make([][]byte, l+1)
	for i := range layerKeys {
		layerKey, _ := keys.GenerateSymmetricKey()
		layerKeys[i] = layerKey
	}
	K, _ := keys.GenerateSymmetricKey()
	masterKey := base64.StdEncoding.EncodeToString(K) // Convert the master key to a base64 string.

	// Construct the first sepal for the onion using the master key.
	A, S, err := onion_model.FormSepals(masterKey, d, layerKeys, l, l1, l2, Hash)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create sepal")
	}

	// Form the content of the onion by encrypting the message for each layer.
	C, err := onion_model.FormContent(layerKeys, l, message, K)
	if err != nil {
		return nil, pl.WrapError(err, "failed to form content")
	}

	// Form the headers for each layer of the onion.
	H, err := onion_model.FormHeaders(l, l1, C, A, publicKeys, recipient, layerKeys, path, Hash, metadata)
	if err != nil {
		return nil, pl.WrapError(err, "failed to form headers")
	}

	// Initialize the onion structure.
	onionLayers := make([][]onion_model.Onion, l)

	// Populate the onion layers with headers, content, and sepals.
	for i := 0; i < len(H)-1; i++ {
		if i < len(S) && S[i] != nil {
			onionLayers[i] = utils.Map(S[i], func(sepal onion_model.Sepal) onion_model.Onion {
				return onion_model.Onion{
					Header:  H[i+1],
					Content: C[i+1],
					Sepal:   sepal,
				}
			})
		} else {
			onionLayers[i] = []onion_model.Onion{{
				Header:  H[i+1],
				Content: C[i+1],
				Sepal:   onion_model.Sepal{Blocks: []string{}},
			}}
		}
	}

	// Return the constructed onion layers.
	return onionLayers, nil
}

// Hash generates a hash of the given string using FNV-1a hashing algorithm.
func Hash(s string) string {
	h := fnv.New32a()            // Initialize a new FNV-1a hash function.
	_, err := h.Write([]byte(s)) // Write the string data to the hash function.
	if err != nil {
		slog.Error("failed to Hash string", err) // Log any errors that occur during hashing.
		return ""
	}
	return fmt.Sprint(h.Sum32()) // Return the resulting hash as a string.
}

// padMessage pads the input message to a fixed length with null characters.
func padMessage(message string) []byte {
	var nullTerminator byte = '\000'
	var paddedMessage = make([]byte, fixedLengthOfMessage) // Initialize a byte slice with the fixed length.
	var mLength = len(message)

	// Copy the message into the padded message buffer and fill the rest with null characters.
	for i := 0; i < fixedLengthOfMessage; i++ {
		if i >= mLength || i == fixedLengthOfMessage-1 {
			paddedMessage[i] = nullTerminator // Add null terminator at the end.
		} else {
			paddedMessage[i] = message[i] // Copy the message content.
		}
	}
	return paddedMessage // Return the padded message.
}
