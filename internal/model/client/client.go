package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/tools/keys"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	ID               int
	Host             string
	Port             int
	Address          string
	PrivateKey       string
	PublicKey        string
	SessionKeys      map[string][]byte
	ActiveNodes      []structs.PublicNodeApi
	OtherClients     []structs.PublicNodeApi
	Messages         []structs.Message
	mu               sync.RWMutex
	BulletinBoardUrl string
	status           *structs.ClientStatus
}

// NewNode creates a new client instance
func NewClient(id int, host string, port int, bulletinBoardUrl string) (*Client, error) {
	if privateKey, publicKey, err := keys.KeyGen(); err != nil {
		return nil, pl.WrapError(err, "node.NewClient(): failed to generate key pair")
	} else {
		c := &Client{
			ID:               id,
			Host:             host,
			Port:             port,
			Address:          fmt.Sprintf("http://%s:%d", host, port),
			PublicKey:        publicKey,
			PrivateKey:       privateKey,
			SessionKeys:      make(map[string][]byte),
			ActiveNodes:      make([]structs.PublicNodeApi, 0),
			BulletinBoardUrl: bulletinBoardUrl,
			Messages:         make([]structs.Message, 0),
			OtherClients:     make([]structs.PublicNodeApi, 0),
			status:           structs.NewClientStatus(id, fmt.Sprintf("http://%s:%d", host, port), publicKey),
		}

		if err2 := c.RegisterWithBulletinBoard(); err2 != nil {
			return nil, pl.WrapError(err2, "%s: failed to register with bulletin board", pl.GetFuncName(id, host, port, bulletinBoardUrl))
		}

		return c, nil
	}
}

// RegisterWithBulletinBoard registers the client with the bulletin board
func (c *Client) RegisterWithBulletinBoard() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if data, err := json.Marshal(structs.PublicNodeApi{
		ID:        c.ID,
		Address:   c.Address,
		PublicKey: c.PublicKey,
	}); err != nil {
		return pl.WrapError(err, "%s: failed to marshal Client info", pl.GetFuncName())
	} else {
		url := c.BulletinBoardUrl + "/registerClient"
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return pl.WrapError(err2, "%s: failed to send POST request to bulletin board", pl.GetFuncName())
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					slog.Error(pl.GetFuncName()+": error closing response body", err2)
				}
			}(resp.Body)
			if resp.StatusCode != http.StatusCreated {
				return pl.NewError("%s: failed to register client, status code: %d, %s", pl.GetFuncName(), resp.StatusCode, resp.Status)
			} else {
				slog.Info("Client registered with bulletin board", "id", c.ID)
			}
			return nil
		}
	}
}

func (c *Client) getRecipient() (string, int) {
	numClients := len(config.GlobalConfig.Clients)

	recipients := utils.Reverse(utils.NewIntArray(1, numClients+1))

	if len(recipients) > 3 {
		utils.DeterministicShuffle(recipients[2:], int64(42))
	}

	sendTo := recipients[c.ID-1]
	recipient := *(utils.Find(config.GlobalConfig.Clients, func(client config.Client) bool {
		return client.ID == sendTo
	}))
	return fmt.Sprintf("http://%s:%d", recipient.Host, recipient.Port), recipient.ID
}

// StartGeneratingMessages continuously generates and sends messages to other clients
func (c *Client) StartGeneratingMessages() {
	slog.Info("Client starting to generate messages", "id", c.ID)

	recipientAddress, recipientId := c.getRecipient()

	messages := []structs.Message{
		structs.NewMessage(c.Address, recipientAddress, fmt.Sprintf("Msg from client(id=%d)", c.ID)),
	}

	if err := c.RegisterIntentToSend(messages); err != nil {
		slog.Error(pl.GetFuncName()+": Error registering intent to send", err)
	} else {
		slog.Info(fmt.Sprintf("Client %d sending to client %d", c.ID, recipientId))
		c.mu.Lock()
		defer c.mu.Unlock()
		c.Messages = messages
	}
}

//var rand = rng.New(rng.NewSource(time.Now().UnixNano()))

// DetermineRoutingPath determines a random routing path of a given length
func DetermineRoutingPath(participants []structs.PublicNodeApi) ([]structs.PublicNodeApi, []structs.PublicNodeApi, error) {

	mixers := make([]structs.PublicNodeApi, config.GlobalConfig.L1)
	for i := range mixers {
		mixers[i] = utils.RandomElement(participants)
	}

	gatekeepers := make([]structs.PublicNodeApi, config.GlobalConfig.L2)
	for i := range gatekeepers {
		gatekeepers[i] = utils.RandomElement(participants)
	}

	return mixers, gatekeepers, nil
}

// formOnions forms the onions for the messages to be sent
func (c *Client) formOnions(start structs.ClientStartRunApi) ([]queuedOnion, error) {
	onions := make([]queuedOnion, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range c.Messages {
		if destination := utils.Find(start.Clients, func(node structs.PublicNodeApi) bool {
			return node.Address == c.Messages[i].To
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

	wg.Wait()
	return onions, nil
}

type queuedOnion struct {
	to    string
	onion onion_model.Onion
}

// processMessage processes a single message to form its onion
func (c *Client) processMessage(msg structs.Message, destination structs.PublicNodeApi, nodes []structs.PublicNodeApi) (onions []queuedOnion, err error) {
	onions = make([]queuedOnion, 0)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, pl.WrapError(err, "failed to marshal message")
	}
	msgString := base64.StdEncoding.EncodeToString(msgBytes)

	mixers, gatekeepers, err := DetermineRoutingPath(nodes)
	if err != nil {
		return nil, pl.WrapError(err, "failed to determine routing path")
	}

	routingPath := append(append(mixers, gatekeepers...), destination)
	publicKeys := utils.Map(routingPath, func(node structs.PublicNodeApi) string {
		return node.PublicKey
	})
	mixersAddr := utils.Map(mixers, func(node structs.PublicNodeApi) string {
		return node.Address
	})
	gatekeepersAddr := utils.Map(gatekeepers, func(node structs.PublicNodeApi) string {
		return node.Address
	})

	metadata := make([]onion_model.Metadata, len(routingPath)+1)
	for i := range metadata {
		metadata[i] = onion_model.Metadata{Nonce: ""}
	}

	o, err := pi_t.FORMONION(c.PrivateKey, msgString, mixersAddr, gatekeepersAddr, destination.Address, publicKeys, metadata, config.GlobalConfig.D)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create onion")
	}

	onions = append(onions, queuedOnion{
		onion: o[0][0],
		to:    mixersAddr[0],
	})

	c.status.AddSent(destination, routingPath, msg)

	return onions, nil
}

// createCheckpointOnions creates checkpoint onions for the routing path
func (c *Client) processCheckpoint(checkpointOnion structs.CheckpointOnion, clients []structs.PublicNodeApi) (onions []queuedOnion, err error) {
	onions = make([]queuedOnion, 0)

	path := utils.Map(checkpointOnion.Path, func(cp structs.Checkpoint) structs.PublicNodeApi {
		return cp.Receiver
	})

	clientReceiver := utils.RandomElement(clients)
	checkpointPublicKeys := append(utils.Map(path, func(node structs.PublicNodeApi) string {
		return node.PublicKey
	}), clientReceiver.PublicKey)

	dummyMsg := structs.NewMessage(c.Address, clientReceiver.Address, "checkpoint onion")
	dummyPayload, err := json.Marshal(dummyMsg)
	if err != nil {
		return nil, pl.WrapError(err, "failed to marshal dummy message")
	}
	mString := base64.StdEncoding.EncodeToString(dummyPayload)

	metadata := utils.Map(checkpointOnion.Path, func(cp structs.Checkpoint) onion_model.Metadata {
		return onion_model.Metadata{
			Nonce: cp.Nonce,
		}
	})

	metadata = utils.InsertAtIndex(metadata, 0, onion_model.Metadata{})
	metadata = append(metadata, onion_model.Metadata{})
	mixersAddr := utils.Map(path[:config.GlobalConfig.L1], func(node structs.PublicNodeApi) string {
		return node.Address
	})
	gatekeepersAddr := utils.Map(path[config.GlobalConfig.L1:], func(node structs.PublicNodeApi) string {
		return node.Address
	})

	o, err := pi_t.FORMONION(c.PrivateKey, mString, mixersAddr, gatekeepersAddr, clientReceiver.Address, checkpointPublicKeys, metadata, config.GlobalConfig.D)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create checkpoint onion")
	}

	onions = append(onions, queuedOnion{
		onion: o[0][0],
		to:    path[0].Address,
	})

	c.status.AddSent(utils.GetLast(path), path, dummyMsg)
	return onions, nil
}

func (c *Client) startRun(start structs.ClientStartRunApi) error {

	slog.Info("Client starting run", "num clients", len(start.Clients), "num relays", len(start.Relays))
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(start.Relays) == 0 {
		return pl.NewError("%s: no participating relays", pl.GetFuncName())
	}
	if len(start.Clients) == 0 {
		return pl.NewError("%s: no participating clients", pl.GetFuncName())
	}

	if !utils.Contains(start.Clients, func(client structs.PublicNodeApi) bool {
		return client.ID == c.ID
	}) {
		return nil
	}

	if toSend, err := c.formOnions(start); err != nil {
		return pl.WrapError(err, "failed to form toSend")
	} else {
		numToSend := len(toSend)

		slog.Info("Client sending onions", "num_onions", numToSend)

		var wg sync.WaitGroup
		wg.Add(numToSend)
		for _, onion := range toSend {
			go func(onion queuedOnion) {
				defer wg.Done()
				if err = api_functions.SendOnion(onion.to, c.Address, onion.onion); err != nil {
					slog.Error("failed to send onions", err)
				}
			}(onion)
		}

		wg.Wait()

		c.Messages = make([]structs.Message, 0)
		return nil
	}
}

func (c *Client) Receive(oApi structs.OnionApi) error {
	if _, _, _, peeled, _, err := pi_t.PeelOnion(oApi.Onion, c.PrivateKey); err != nil {
		return pl.WrapError(err, "node.Receive(): failed to remove layer")
	} else {
		slog.Info("Client received onion", "bruises", peeled)

		var msg structs.Message
		if err2 := json.Unmarshal([]byte(peeled.Content), &msg); err2 != nil {
			return pl.WrapError(err2, "node.Receive(): failed to unmarshal message")
		}
		slog.Info("Received message", "from", msg.From, "to", msg.To, "msg", msg.Msg)

		c.status.AddReceived(msg)

	}
	return nil
}

func (c *Client) GetStatus() string {
	return c.status.GetStatus()
}

func (c *Client) RegisterIntentToSend(messages []structs.Message) error {

	//	slog.Info("Client registering intent to send", "id", c.ID, "num_messages", len(messages))

	to := utils.Map(messages, func(m structs.Message) structs.PublicNodeApi {
		if f := utils.Find(c.OtherClients, func(c structs.PublicNodeApi) bool {
			return c.Address == m.To
		}); f != nil {
			return *f
		} else {
			return structs.PublicNodeApi{}
		}
	})
	if data, err := json.Marshal(structs.IntentToSend{
		From: structs.PublicNodeApi{
			ID:        c.ID,
			Address:   c.Address,
			PublicKey: c.PublicKey,
			Time:      time.Now(),
		},
		To: to,
	}); err != nil {
		return pl.WrapError(err, "%s: failed to marshal Client info", pl.GetFuncName())
	} else {
		url := c.BulletinBoardUrl + "/registerIntentToSend"
		//slog.Info("Sending Client registration request.", "url", url, "id", c.ID)
		if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return pl.WrapError(err2, "%s: failed to send POST request to bulletin board", pl.GetFuncName())
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					fmt.Printf("Client.UpdateBulletinBoard(): error closing response body: %v\n", err2)
				}
			}(resp.Body)
			if resp.StatusCode != http.StatusOK {
				return pl.NewError("%s failed to register intent to send, status code: %d, %s", pl.GetFuncName(), resp.StatusCode, resp.Status)
			} else {
				//slog.Info("Client registered intent to send with bulletin board", "id", c.ID)
				c.Messages = messages
			}
			return nil
		}
	}
}
