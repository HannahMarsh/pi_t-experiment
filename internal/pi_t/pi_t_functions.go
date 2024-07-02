package pi_t

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/keys"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/prf"
	"golang.org/x/exp/slog"
	"strings"
)

// OnionPayload represents the actual payload of an onion layer.
// It is encrypted as the CombinedPayload.EncryptedPayload
type OnionPayload struct {
	IsCheckpointOnion bool   // Indicates if this is a checkpoint onion
	Layer             int    // The current layer index
	Nonce             string // The nonce for the layer
	LastHop           string // The previous hop in the route
	NextHop           string // The next hop in the route
	Payload           string // The encrypted payload
	NextHopPubKey     string // The public key of the next hop
}

// CombinedPayload represents the combined encrypted payload structure.
// It includes the shared key from the original sender (client)
type CombinedPayload struct {
	EncryptedSharedKey string // The encrypted shared key
	EncryptedPayload   string // The encrypted payload
	ClientPubKey       string // The original sender's public key
	ClientScalar       string // The original sender's scalar
}

// Header represents the header of the onion with the bruise counter.
// It includes the shared key from the previous hop (node)
type Header struct {
	BruiseCounter            int    // The bruise counter
	EncryptedSharedKey       string // The encrypted shared key
	SenderPubKey             string // The sender's public key
	CombinedEncryptedPayload string // The combined encrypted payload
}

// FormOnion creates an onion by encapsulating a message in multiple encryption layers.
// Parameters:
// - privateKeyPEM: The PEM-encoded private key of the node.
// - publicKeyPEM: The PEM-encoded public key of the sender.
// - payload: The plaintext payload to be encapsulated.
// - publicKeys: A slice of PEM-encoded public keys for the nodes in the route.
// - routingPath: A slice of node identifiers in the route.
// - checkpoint: The index of the checkpoint layer.
// Returns:
// - The destination node identifier.
// - The base64-encoded onion with the added header.
// - An error object if an error occurred, otherwise nil.
func FormOnion(privateKeyPEM string, publicKeyPEM string, payload []byte, publicKeys []string, routingPath []string, checkpoint int, clientAddr string) (string, string, []bool, error) {
	var sendCheckPoints []bool

	if checkpoint <= 0 {
		sendCheckPoints = make([]bool, len(publicKeys))
		for i := range sendCheckPoints {
			sendCheckPoints[i] = false
		}
	}

	for i := len(publicKeys) - 1; i >= 0; i-- {
		var layerBytes []byte
		var err error

		scalar, err := keys.GenerateScalar()
		if err != nil {
			return "", "", nil, pl.WrapError(err, "failed to generate scalar")
		}

		if i == len(publicKeys)-1 {
			layerBytes = payload
		} else {
			lastHop := clientAddr
			if i > 0 {
				lastHop = routingPath[i-1]
			}
			nextHop := ""
			if len(routingPath) >= i+2 {
				nextHop = routingPath[i+1]
			}
			layer := OnionPayload{
				IsCheckpointOnion: checkpoint == i,
				Layer:             i,
				LastHop:           lastHop,
				NextHop:           nextHop,
				Payload:           base64.StdEncoding.EncodeToString(payload),
				NextHopPubKey:     publicKeys[i+1],
			}

			var nonce []byte
			if checkpoint <= 0 {
				// Use PRF_F1 to determine if a checkpoint onion is expected
				checkpointExpected := prf.PRF_F1(privateKeyPEM, publicKeys[i], scalar, i)
				if checkpointExpected == 0 {
					sendCheckPoints[i] = true
				}
				if checkpointExpected == 0 {
					nonce = prf.PRF_F2(privateKeyPEM, publicKeys[i], scalar, i)
				}
			}
			if nonce == nil {
				nonce = make([]byte, 16)
				_, err = rand.Read(nonce)
				if err != nil {
					return "", "", nil, pl.WrapError(err, "failed to generate random nonce")
				}
			}

			layer.Nonce = base64.StdEncoding.EncodeToString(nonce)

			layerBytes, err = json.Marshal(layer)
			if err != nil {
				return "", "", nil, pl.WrapError(err, "failed to marshal onion layer")
			}
		}

		encryptedSharedKey, encryptedPayload, err := keys.EncodeWithScalar(layerBytes, privateKeyPEM, publicKeys[i], scalar)

		combinedPayload := CombinedPayload{
			EncryptedSharedKey: encryptedSharedKey, //base64.StdEncoding.EncodeToString([]byte(encryptedKey)),
			EncryptedPayload:   encryptedPayload,
			ClientPubKey:       publicKeyPEM,
			ClientScalar:       base64.StdEncoding.EncodeToString(scalar),
		}

		payload, err = json.Marshal(combinedPayload)
		if err != nil {
			return "", "", nil, pl.WrapError(err, "failed to marshal combined payload")
		}
	}

	onionWithHeader, err := addHeaderAfterPeeling(base64.StdEncoding.EncodeToString(payload), privateKeyPEM, publicKeyPEM, publicKeys[0], 0)
	if err != nil {
		return "", "", nil, pl.WrapError(err, "failed to add header")
	}

	return routingPath[0], onionWithHeader, sendCheckPoints, nil
}

// AddHeader adds a header to the peeled onion payload.
// Parameters:
// - peeledOnion: The peeled onion payload.
// - bruiseCounter: The bruise counter to be added.
// - privateKeyPEM: The PEM-encoded private key of the node.
// - senderPubicKey: The PEM-encoded public key of the sender.
// Returns:
// - The base64-encoded onion with the added header.
// - An error object if an error occurred, otherwise nil.
func AddHeader(peeledOnion *OnionPayload, bruiseCounter int, privateKeyPEM string, senderPubicKey string) (string, error) {
	return addHeaderAfterPeeling(peeledOnion.Payload, privateKeyPEM, senderPubicKey, peeledOnion.NextHopPubKey, bruiseCounter)
}

// addHeaderAfterPeeling adds a header to the peeled onion payload after decryption.
// Parameters:
// - payload: The base64-encoded peeled onion payload.
// - privateKeyPEM: The PEM-encoded private key of the node.
// - senderPubicKey: The PEM-encoded public key of the sender.
// - receiverPublicKey: The PEM-encoded public key of the receiver.
// - bruiseCounter: The bruise counter to be added.
// Returns:
// - The base64-encoded onion with the added header.
// - An error object if an error occurred, otherwise nil.
func addHeaderAfterPeeling(payload string, privateKeyPEM string, senderPubicKey string, receiverPublicKey string, bruiseCounter int) (string, error) {
	payloadbytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", pl.WrapError(err, "failed to decode payload")
	}
	encryptedSharedKey, encryptedPayload, err := keys.Enc(payloadbytes, privateKeyPEM, receiverPublicKey)
	if err != nil {
		return "", pl.WrapError(err, "failed to encrypt payload")
	}
	header := Header{
		BruiseCounter:            bruiseCounter,
		EncryptedSharedKey:       encryptedSharedKey,
		SenderPubKey:             senderPubicKey,
		CombinedEncryptedPayload: encryptedPayload,
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", pl.WrapError(err, "failed to marshal header")
	}
	return base64.StdEncoding.EncodeToString(headerBytes), nil
}

// removeHeader removes the header from the onion and decrypts the payload.
// Parameters:
// - onion: The base64-encoded onion with the header.
// - privateKeyPEM: The PEM-encoded private key of the node.
// Returns:
// - The base64-encoded payload after removing the header.
// - The bruise counter.
// - An error object if an error occurred, otherwise nil.
func removeHeader(onion string, privateKeyPEM string) (string, int, error) {
	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return "", 0, pl.WrapError(err, "failed to decode onion")
	}
	var header Header
	if err = json.Unmarshal(onionBytes, &header); err != nil {
		return "", 0, pl.WrapError(err, "failed to unmarshal header")
	}
	decryptedPayload, err := keys.Dec(header.EncryptedSharedKey, header.CombinedEncryptedPayload, privateKeyPEM, header.SenderPubKey)
	if err != nil {
		return "", 0, pl.WrapError(err, "failed to decrypt payload")
	}
	return base64.StdEncoding.EncodeToString(decryptedPayload), header.BruiseCounter, nil
}

// PeelOnion removes the outermost layer of the onion.
// Parameters:
// - onion: The base64-encoded onion.
// - privateKeyPEM: The PEM-encoded private key of the node.
// Returns:
// - The peeled onion payload.
// - The bruise counter.
// - A boolean indicating if the nonce verification passed.
// - A boolean indicating if we should expect a checkpoint onion this round
// - An error object if an error occurred, otherwise nil.
func PeelOnion(onion string, privateKeyPEM string) (*OnionPayload, int, bool, bool, error) {
	headerRemoved, bruises, err := removeHeader(onion, privateKeyPEM)
	if err != nil {
		return nil, -1, true, false, pl.WrapError(err, "failed to remove header")
	}

	peeled, nonceVerification, expectCheckpoint, err := peelOnionAfterRemovingPayload(headerRemoved, privateKeyPEM)
	if err != nil {
		return nil, -1, nonceVerification, expectCheckpoint, pl.WrapError(err, "failed to peel onion")
	}

	return peeled, bruises, nonceVerification, !peeled.IsCheckpointOnion && expectCheckpoint, nil
}

// peelOnionAfterRemovingPayload removes the outermost layer of the onion after removing the header.
// Parameters:
// - onion: The base64-encoded onion without the header.
// - privateKeyPEM: The PEM-encoded private key of the node.
// Returns:
// - The peeled onion payload.
// - A boolean indicating if the nonce verification passed.
// - An error object if an error occurred, otherwise nil.
func peelOnionAfterRemovingPayload(onion string, privateKeyPEM string) (*OnionPayload, bool, bool, error) {

	onionBytes, err := base64.StdEncoding.DecodeString(onion)
	if err != nil {
		return nil, true, false, pl.WrapError(err, "failed to decode onion")
	}

	var combinedPayload CombinedPayload
	if err = json.Unmarshal(onionBytes, &combinedPayload); err != nil {
		return nil, true, false, pl.WrapError(err, "failed to unmarshal combined payload")
	}

	scalar, err := base64.StdEncoding.DecodeString(combinedPayload.ClientScalar)
	if err != nil {
		return nil, true, false, pl.WrapError(err, "failed to decode scalar")
	}

	decryptedBytes, err := keys.DecodeWithScalar(combinedPayload.EncryptedSharedKey, combinedPayload.EncryptedPayload, privateKeyPEM, combinedPayload.ClientPubKey, scalar)

	decryptedPayload := string(decryptedBytes)

	if !strings.HasPrefix(decryptedPayload, "{\"IsCheckpointOnion\":") {
		return &OnionPayload{
			IsCheckpointOnion: false,
			Layer:             0,
			NextHop:           "",
			LastHop:           "",
			Payload:           decryptedPayload,
		}, true, false, nil
	}

	var layer OnionPayload
	err = json.Unmarshal(decryptedBytes, &layer)
	if err != nil {
		return nil, true, false, pl.WrapError(err, "failed to unmarshal onion layer")
	}

	nonce, err := base64.StdEncoding.DecodeString(layer.Nonce)
	if err != nil {
		return nil, true, false, pl.WrapError(err, "failed to decode nonce")
	}

	checkpointExpected := prf.PRF_F1(privateKeyPEM, combinedPayload.ClientPubKey, scalar, layer.Layer)
	if checkpointExpected == 0 {
		expectedNonce := prf.PRF_F2(privateKeyPEM, combinedPayload.ClientPubKey, scalar, layer.Layer)
		if !hmac.Equal(nonce, expectedNonce) {
			slog.Warn("nonce verification failed")
			return &layer, false, true, nil
		}
	}

	return &layer, true, checkpointExpected == 0, nil
}
