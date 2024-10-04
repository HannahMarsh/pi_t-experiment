package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/metrics"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Client represents a user in the network.
type Client struct {
	ID               int                     // Unique identifier for the client.
	Host             string                  // Host address of the client.
	Port             int                     // Port number on which the client listens.
	Address          string                  // Full address of the client in the form http://host:port.
	PrivateKey       string                  // Client's long term private key for decryption.
	PublicKey        string                  // Client's long term public key for encryption.
	PrometheusPort   int                     // Port number for Prometheus metrics.
	ActiveRelays     []structs.PublicNodeApi // List of active relays known to the client.
	OtherClients     []structs.PublicNodeApi // List of other client known to the client.
	Messages         []structs.Message       // Messages to be sent by the client.
	BulletinBoardUrl string                  // URL of the bulletin board for client registration and communication.
	status           *structs.ClientStatus   // Client status, including sent and received messages.
	wg               sync.WaitGroup          // WaitGroup to ensure the client does not start protocol until all messages are generated
	mu               sync.RWMutex
}

// NewClient creates a new client instance with a unique ID, host, and port.
func NewClient(id int, host string, port int, promPort int, bulletinBoardUrl string) (*Client, error) {
	// Generate a key pair (private and public) for the client.
	if privateKey, publicKey, err := keys.KeyGen(); err != nil {
		return nil, pl.WrapError(err, "relay.NewClient(): failed to generate key pair")
	} else {
		c := &Client{
			ID:               id,
			Host:             host,
			Port:             port,
			Address:          fmt.Sprintf("http://%s:%d", host, port),
			PublicKey:        publicKey,
			PrivateKey:       privateKey,
			PrometheusPort:   promPort,
			ActiveRelays:     make([]structs.PublicNodeApi, 0),
			BulletinBoardUrl: bulletinBoardUrl,
			Messages:         make([]structs.Message, 0),
			OtherClients:     make([]structs.PublicNodeApi, 0),
			status:           structs.NewClientStatus(id, port, promPort, fmt.Sprintf("http://%s:%d", host, port), host, publicKey),
		}
		c.wg.Add(1)

		// Register the client with the bulletin board.
		if err := c.RegisterWithBulletinBoard(); err != nil {
			return nil, pl.WrapError(err, "%s: failed to register with bulletin board", pl.GetFuncName(id, host, port, bulletinBoardUrl))
		}

		return c, nil
	}
}

// RegisterWithBulletinBoard registers the client with the bulletin board by sending its public information.
func (c *Client) RegisterWithBulletinBoard() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Marshal the client's public information into JSON.
	if data, err := json.Marshal(structs.PublicNodeApi{
		ID:             c.ID,
		Address:        c.Address,
		PublicKey:      c.PublicKey,
		PrometheusPort: c.PrometheusPort,
		Host:           c.Host,
		Port:           c.Port,
	}); err != nil {
		return pl.WrapError(err, "%s: failed to marshal Client info", pl.GetFuncName())
	} else {
		// Send a POST request to the bulletin board to register the client.
		url := c.BulletinBoardUrl + "/registerClient"
		if resp, err := http.Post(url, "application/json", bytes.NewBuffer(data)); err != nil {
			return pl.WrapError(err, "%s: failed to send POST request to bulletin board", pl.GetFuncName())
		} else {
			defer func(Body io.ReadCloser) {
				// avoid resource leaks.
				if err := Body.Close(); err != nil {
					slog.Error(pl.GetFuncName()+": error closing response body", err)
				}
			}(resp.Body)
			// Check if the client was registered successfully
			if resp.StatusCode != http.StatusCreated {
				return pl.NewError("%s: failed to register client, status code: %d, %s", pl.GetFuncName(), resp.StatusCode, resp.Status)
			} else {
				slog.Info("Client registered with bulletin board", "id", c.ID)
			}
			return nil
		}
	}
}

// getRecipient determines the recipient client for sending a message based on the client's ID.
func (c *Client) getRecipient(clients []structs.PublicNodeApi) (string, int) {
	numClients := len(clients)

	// Generate a reversed array of client IDs.
	recipients := utils.Reverse(utils.NewIntArray(1, numClients+1))

	// If there are more than 3 recipients, shuffle the IDs deterministically.
	if len(recipients) > 3 {
		utils.DeterministicShuffle(recipients[2:], int64(42))
	}

	// Determine the recipient ID based on the client's ID.
	sendTo := recipients[c.ID-1]
	recipient := *(utils.Find(clients, func(client structs.PublicNodeApi) bool {
		return client.ID == sendTo
	}))
	return fmt.Sprintf("http://%s:%d", recipient.Host, recipient.Port), recipient.ID // Return the recipient's address and ID.
}

// StartGeneratingMessages generates a single message to be sent to another client.
func (c *Client) generateMessages(start structs.ClientStartRunApi) {
	defer c.wg.Done() // Mark this operation as done in the WaitGroup when finished.
	slog.Info("Client starting to generate messages", "id", c.ID)

	// Get the recipient's address and ID.
	recipientAddress, _ := c.getRecipient(start.Clients)

	// Create a new message to send to the recipient.
	messages := []structs.Message{
		structs.NewMessage(c.Address, recipientAddress, fmt.Sprintf("Msg from client(id=%d)", c.ID)),
	}

	// Register the intent to send the message with the bulletin board.
	//if err := c.RegisterIntentToSend(messages); err != nil {
	//	slog.Error(pl.GetFuncName()+": Error registering intent to send", err)
	//} else {
	//	slog.Info(fmt.Sprintf("Client %d sending to client %d", c.ID, recipientId))
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = messages // Store the messages to be sent.
	//}
}

// DetermineRoutingPath determines a random routing path of mixers and gatekeepers.
func DetermineRoutingPath(participants []structs.PublicNodeApi) ([]structs.PublicNodeApi, []structs.PublicNodeApi, error) {
	// initialize slices for mixers and gatekeepers in the path.
	mixers := make([]structs.PublicNodeApi, config.GetL1())
	gatekeepers := make([]structs.PublicNodeApi, config.GetL2())

	for i := range mixers {
		mixers[i] = utils.RandomElement(participants) // Randomly select a mixer for each layer.
	}
	for i := range gatekeepers {
		gatekeepers[i] = utils.RandomElement(participants) // Randomly select a gatekeeper for each layer.
	}

	return mixers, gatekeepers, nil // Return the slices of mixers and gatekeepers.
}

// formOnions forms the onions for the messages to be sent by the client.
func (c *Client) formOnions(start structs.ClientStartRunApi) ([]queuedOnion, error) {
	onions := make([]queuedOnion, 0) // Initialize a slice to hold the formed onions.

	var mu sync.Mutex     // Mutex to ensure concurrent access safety while forming onions.
	var wg sync.WaitGroup // WaitGroup to manage concurrent onion formation.

	// Iterate over the client's messages to form onions for each one.
	for i := range c.Messages {
		if destination := utils.Find(start.Clients, func(relay structs.PublicNodeApi) bool {
			return relay.Address == c.Messages[i].To
		}); destination != nil {
			wg.Add(1)
			go func(destination structs.PublicNodeApi) {
				defer wg.Done()
				if o, err := c.processMessage(c.Messages[i], destination, start.Relays); err != nil {
					slog.Error("failed to process message", err)
				} else {
					mu.Lock()
					onions = append(onions, o...)
					mu.Unlock()
				}
			}(*destination)
		}
	}

	// Iterate over the client's checkpoint onions to form them.
	for _, checkpointOnion := range start.CheckpointOnions {
		wg.Add(1)
		go func(checkpointOnion structs.CheckpointOnion) {
			defer wg.Done()
			if o, err := c.processCheckpoint(checkpointOnion, start.Clients); err != nil {
				slog.Error("failed to process checkpoint", err)
			} else {
				mu.Lock()
				onions = append(onions, o...)
				mu.Unlock()
			}
		}(checkpointOnion)
	}

	wg.Wait()          // Wait for all onion formation operations to complete.
	return onions, nil // Return the formed onions.
}

// queuedOnion represents an onion that is ready to be sent, including its destination and the message it encapsulates.
type queuedOnion struct {
	to    string            // The address to which the onion should be sent.
	onion onion_model.Onion // The onion itself.
	msg   structs.Message   // The original message that the onion encapsulates.
}

// processMessage processes a single message to form its onion.
func (c *Client) processMessage(msg structs.Message, destination structs.PublicNodeApi, relays []structs.PublicNodeApi) (onions []queuedOnion, err error) {
	onions = make([]queuedOnion, 0)    // Initialize a slice to hold the formed onions for this message.
	msgBytes, err := json.Marshal(msg) // Marshal the message into JSON.
	if err != nil {
		return nil, pl.WrapError(err, "failed to marshal message")
	}

	// Determine the routing path (mixers and gatekeepers) for this message.
	mixers, gatekeepers, err := DetermineRoutingPath(relays)
	if err != nil {
		return nil, pl.WrapError(err, "failed to determine routing path")
	}

	// Combine the mixers and gatekeepers with the final destination to form the complete routing path.
	routingPath := append(append(mixers, gatekeepers...), destination)
	publicKeys := utils.Map(routingPath, func(node structs.PublicNodeApi) string {
		return node.PublicKey
	})
	mixersAddr := utils.Map(mixers, func(mixer structs.PublicNodeApi) string {
		return mixer.Address
	})
	gatekeepersAddr := utils.Map(gatekeepers, func(gatekeeper structs.PublicNodeApi) string {
		return gatekeeper.Address
	})

	// Prepare the metadata for each layer in the onion.
	metadata := make([]onion_model.Metadata, len(routingPath)+1)
	for i := range metadata {
		metadata[i] = onion_model.Metadata{Nonce: ""}
	}

	// Form the onion using the client's private key and the determined routing path.
	o, err := pi_t.FORMONION(string(msgBytes), mixersAddr, gatekeepersAddr, destination.Address, publicKeys, metadata, config.GetD())
	if err != nil {
		return nil, pl.WrapError(err, "failed to create onion")
	}

	// Add the formed onion to the list of onions to be sent.
	onions = append(onions, queuedOnion{
		onion: o[0][0],
		to:    mixersAddr[0],
		msg:   msg,
	})

	// Record the sent message in the client's status.
	c.status.AddSent(destination, routingPath, msg)

	return onions, nil // Return the formed onions.
}

// processCheckpoint processes a checkpoint onion for a given routing path.
func (c *Client) processCheckpoint(checkpointOnion structs.CheckpointOnion, clients []structs.PublicNodeApi) (onions []queuedOnion, err error) {
	onions = make([]queuedOnion, 0) // Initialize a slice to hold the formed checkpoint onions.

	// Extract the routing path from the checkpoint onion.
	path := utils.Map(checkpointOnion.Path, func(cp structs.Checkpoint) structs.PublicNodeApi {
		return cp.Receiver
	})

	// Randomly select a client as the final receiver for the checkpoint onion.
	clientReceiver := utils.RandomElement(clients)
	checkpointPublicKeys := append(utils.Map(path, func(node structs.PublicNodeApi) string {
		return node.PublicKey
	}), clientReceiver.PublicKey)

	// Create a dummy message to be encapsulated in the checkpoint onion.
	dummyMsg := structs.NewMessage(c.Address, clientReceiver.Address, "")
	dummyPayload, err := json.Marshal(dummyMsg)
	if err != nil {
		return nil, pl.WrapError(err, "failed to marshal dummy message")
	}

	// Prepare the metadata for each layer in the checkpoint onion.
	metadata := utils.Map(checkpointOnion.Path, func(cp structs.Checkpoint) onion_model.Metadata {
		return onion_model.Metadata{
			Nonce: cp.Nonce,
		}
	})
	metadata = utils.InsertAtIndex(metadata, 0, onion_model.Metadata{})
	metadata = append(metadata, onion_model.Metadata{})

	// Extract the addresses of mixers and gatekeepers for the routing path.
	mixersAddr := utils.Map(path[:config.GetL1()], func(mixer structs.PublicNodeApi) string {
		return mixer.Address
	})
	gatekeepersAddr := utils.Map(path[config.GetL1():], func(gatekeeper structs.PublicNodeApi) string {
		return gatekeeper.Address
	})

	// Form the checkpoint onion using the client's private key and the determined routing path.
	o, err := pi_t.FORMONION(string(dummyPayload), mixersAddr, gatekeepersAddr, clientReceiver.Address, checkpointPublicKeys, metadata, config.GetD())
	if err != nil {
		return nil, pl.WrapError(err, "failed to create checkpoint onion")
	}

	// Add the formed checkpoint onion to the list of onions to be sent.
	onions = append(onions, queuedOnion{
		onion: o[0][0],
		to:    path[0].Address,
		msg:   dummyMsg,
	})

	// Record the sent dummy message in the client's status.
	c.status.AddSent(utils.GetLast(path), path, dummyMsg)
	return onions, nil // Return the formed checkpoint onions.
}

// startRun initiates a communication run based on the start signal received from the bulletin board.
func (c *Client) startRun(start structs.ClientStartRunApi) error {
	slog.Info("Client starting run", "num client", len(start.Clients), "num relays", len(start.Relays))
	c.mu.Lock()         // Lock the mutex to ensure exclusive access to the client's state during the run.
	defer c.mu.Unlock() // Unlock the mutex when the function returns.

	config.UpdateConfig(start.Config) // Update the global configuration based on the start signal.

	c.generateMessages(start)

	// Ensure that there are relays and client participating in the run.
	if len(start.Relays) == 0 {
		return pl.NewError("%s: no participating relays", pl.GetFuncName())
	}
	if len(start.Clients) == 0 {
		return pl.NewError("%s: no participating client", pl.GetFuncName())
	}

	// Check if the current client is included in the list of participating client.
	if !utils.Contains(start.Clients, func(client structs.PublicNodeApi) bool {
		return client.ID == c.ID
	}) {
		return nil // If the client is not included, return without doing anything.
	}

	// Form the onions for the messages to be sent during the run.
	if toSend, err := c.formOnions(start); err != nil {
		return pl.WrapError(err, "failed to form toSend")
	} else {
		numToSend := len(toSend) // Get the number of onions to send.

		slog.Info("Client sending onions", "num_onions", numToSend)

		var wg sync.WaitGroup // WaitGroup to manage concurrent sending of onions.
		wg.Add(numToSend)
		for _, onion := range toSend {
			go func(onion queuedOnion) {
				defer wg.Done()
				timeSent := time.Now()
				if err = api_functions.SendOnion(onion.to, c.Address, onion.onion, 0); err != nil {
					slog.Error("failed to send onions", err)
				}
				metrics.Set(metrics.MSG_SENT, float64(timeSent.Unix()), onion.msg.Hash) // Record the time when the onion was sent.

			}(onion)
		}

		wg.Wait() // Wait for all onions to be sent.

		c.Messages = make([]structs.Message, 0) // Clear the client's messages after sending.
		return nil
	}
}

// Receive processes an incoming onion, decrypts it, and extracts the encapsulated message.
func (c *Client) Receive(oApi structs.OnionApi) error {
	timeReceived := time.Now() // Record the time when the onion was received.
	_, layer, _, peeled, _, err := pi_t.PeelOnion(oApi.Onion, c.PrivateKey)
	if err != nil {
		return pl.WrapError(err, "relay.Receive(): failed to remove layer")
	}

	var msg structs.Message
	if err2 := json.Unmarshal([]byte(peeled.Content), &msg); err2 != nil {
		return pl.WrapError(err2, "relay.Receive(): failed to unmarshal message")
	}
	slog.Info("Client received onion", "layer", layer, "from", msg.From, "message", msg.Msg)

	// Record the received message in the client's status.
	c.status.AddReceived(msg)
	metrics.Set(metrics.MSG_RECEIVED, float64(timeReceived.Unix()), msg.Hash) // Record the time when the message was received.

	return nil
}

// GetStatus returns the current status of the client, including sent and received messages.
func (c *Client) GetStatus() string {
	return c.status.GetStatus()
}

// RegisterIntentToSend registers the client's intent to send messages with the bulletin board.
//func (c *Client) RegisterIntentToSend(messages []structs.Message) error {
//	// Convert the list of messages into a list of public node APIs for the recipients.
//	to := utils.Map(messages, func(m structs.Message) structs.PublicNodeApi {
//		if f := utils.Find(c.OtherClients, func(c structs.PublicNodeApi) bool {
//			return c.Address == m.To
//		}); f != nil {
//			return *f
//		} else {
//			return structs.PublicNodeApi{}
//		}
//	})
//
//	// Marshal the intent-to-send data into JSON.
//	if data, err := json.Marshal(structs.IntentToSend{
//		From: structs.PublicNodeApi{
//			ID:             c.ID,
//			Address:        c.Address,
//			PublicKey:      c.PublicKey,
//			Host:           c.Host,
//			Port:           c.Port,
//			PrometheusPort: c.PrometheusPort,
//			Time:           time.Now(),
//		},
//		To: to,
//	}); err != nil {
//		return pl.WrapError(err, "%s: failed to marshal Client info", pl.GetFuncName())
//	} else {
//		// Send a POST request to the bulletin board to register the intent to send messages.
//		url := c.BulletinBoardUrl + "/registerIntentToSend"
//		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
//			return pl.WrapError(err2, "%s: failed to send POST request to bulletin board", pl.GetFuncName())
//		} else {
//			defer func(Body io.ReadCloser) {
//				// Ensure the response body is closed to avoid resource leaks.
//				if err3 := Body.Close(); err3 != nil {
//					fmt.Printf("Client.UpdateBulletinBoard(): error closing response body: %v\n", err2)
//				}
//			}(resp.Body)
//			// Check if the intent to send was registered successfully based on the HTTP status code.
//			if resp.StatusCode != http.StatusOK {
//				return pl.NewError("%s failed to register intent to send, status code: %d, %s", pl.GetFuncName(), resp.StatusCode, resp.Status)
//			} else {
//				c.Messages = messages // Store the messages to be sent.
//			}
//			return nil
//		}
//	}
//}
