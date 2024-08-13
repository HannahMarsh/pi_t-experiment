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
	rng "math/rand"
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

// StartGeneratingMessages continuously generates and sends messages to other clients
func (c *Client) StartGeneratingMessages(client_addresses []string) {
	slog.Info("Client starting to generate messages", "id", c.ID)
	msgNum := 0
	numMessages := 2
	for i := 0; i < numMessages; i++ {
		select {
		case <-config.GlobalCtx.Done():
			slog.Info(pl.GetFuncName()+": ctx.done -> Client stopping to generate messages", "id", c.ID)
			return
		default:
			messages := c.generateMessages(client_addresses, msgNum)
			msgNum = msgNum + len(messages)

			c.mu.Lock()
			messages = append(c.Messages, messages...)
			if err := c.RegisterIntentToSend(messages); err != nil {
				slog.Error(pl.GetFuncName()+": Error registering intent to send", err)
			} else {
				c.Messages = messages
			}
			c.mu.Unlock()
		}
		//time.Sleep(1 * time.Second)
	}
}

// generateMessages creates messages to be sent to other clients
func (c *Client) generateMessages(client_addresses []string, msgNum int) []structs.Message {
	messages := make([]structs.Message, 0)
	for _, addr := range client_addresses {
		if addr != c.Address && addr != "" {
			messages = append(messages, structs.NewMessage(c.Address, addr, fmt.Sprintf("Msg#%d from client(id=%d)", msgNum, c.ID)))
			msgNum = msgNum + 1
		}
	}
	return messages
}

var rand = rng.New(rng.NewSource(time.Now().UnixNano()))

// DetermineRoutingPath determines a random routing path of a given length
func DetermineRoutingPath(l1, l2 int, participants []structs.PublicNodeApi) ([]structs.PublicNodeApi, []structs.PublicNodeApi, error) {
	mixers := utils.Filter(participants, func(node structs.PublicNodeApi) bool {
		return node.IsMixer
	})

	gateKeepers := utils.Filter(participants, func(node structs.PublicNodeApi) bool {
		return !node.IsMixer
	})

	selectedMixers := make([]structs.PublicNodeApi, l1)
	perm := rand.Perm(len(mixers))

	for i := 0; i < l1; i++ {
		selectedMixers[i] = mixers[perm[i]]
	}

	selectedGateKeepers := make([]structs.PublicNodeApi, l2)
	perm = rand.Perm(len(gateKeepers))

	for i := 0; i < l2; i++ {
		selectedGateKeepers[i] = gateKeepers[perm[i]]
	}

	return selectedMixers, selectedGateKeepers, nil
}

// DetermineCheckpointRoutingPath determines a routing path with a checkpoint
func DetermineCheckpointRoutingPath(checkpoint structs.Checkpoint, nodes []structs.PublicNodeApi, participatingClients []structs.PublicNodeApi) ([]structs.PublicNodeApi, []structs.PublicNodeApi, structs.PublicNodeApi, error) {

	checkpointReceiver := checkpoint.Receiver
	round := checkpoint.Layer
	l1 := config.GlobalConfig.L1
	l2 := config.GlobalConfig.L2

	if checkpointReceiver.IsMixer {
		if round > l1 {
			return nil, nil, structs.PublicNodeApi{}, pl.NewError("round > l1")
		}
		l1 = l1 - 1
	} else {
		if round <= l1 {
			return nil, nil, structs.PublicNodeApi{}, pl.NewError("round <= l1")
		}
		l2 = l2 - 1
	}
	mixers, gatekeepers, err := DetermineRoutingPath(l1, l2, utils.Remove(nodes, func(p structs.PublicNodeApi) bool {
		return p.Address == checkpointReceiver.Address
	}))
	if err != nil {
		return nil, nil, structs.PublicNodeApi{}, pl.WrapError(err, "failed to determine routing path")
	}

	rel := utils.RandomElement(participatingClients)
	if checkpointReceiver.IsMixer {
		return utils.InsertAtIndex(mixers, round, checkpointReceiver), gatekeepers, rel, nil
	} else {
		return mixers, utils.InsertAtIndex(gatekeepers, (round-l1)-1, checkpointReceiver), rel, nil
	}
}

// formOnions forms the onions for the messages to be sent
func (c *Client) formOnions(start structs.ClientStartRunApi) ([]queuedOnion, error) {
	onions := make([]queuedOnion, 0)
	nodes := utils.Filter(append(utils.Copy(start.Mixers), utils.Copy(start.Gatekeepers)...), func(node structs.PublicNodeApi) bool {
		return node.Address != c.Address && node.Address != ""
	})
	clients := utils.Filter(start.Clients, func(node structs.PublicNodeApi) bool {
		return node.Address != c.Address && node.Address != ""
	})

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range c.Messages {
		if destination := utils.Find(start.Clients, func(node structs.PublicNodeApi) bool {
			return node.Address == c.Messages[i].To
		}); destination != nil {
			wg.Add(1)
			go func(destination structs.PublicNodeApi) {
				defer wg.Done()
				if o, err := c.processMessage(c.Messages[i], destination, nodes); err != nil {
					slog.Error("failed to process message", err)
				} else {
					mu.Lock()
					onions = append(onions, o...)
					mu.Unlock()
				}
			}(*destination)
		}
	}

	for _, checkpoint := range start.Checkpoints {
		wg.Add(1)
		go func(checkpoint structs.Checkpoint) {
			defer wg.Done()
			if o, err := c.processCheckpoint(checkpoint, nodes, clients); err != nil {
				slog.Error("failed to process checkpoint", err)
			} else {
				mu.Lock()
				onions = append(onions, o...)
				mu.Unlock()
			}
		}(checkpoint)
	}

	wg.Wait()
	return onions, nil
}

type queuedOnion struct {
	to        string
	onion     onion_model.Onion
	sharedKey string // TODO remove this
}

// processMessage processes a single message to form its onion
func (c *Client) processMessage(msg structs.Message, destination structs.PublicNodeApi, nodes []structs.PublicNodeApi) (onions []queuedOnion, err error) {
	onions = make([]queuedOnion, 0)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, pl.WrapError(err, "failed to marshal message")
	}
	msgString := base64.StdEncoding.EncodeToString(msgBytes)

	mixers, gatekeepers, err := DetermineRoutingPath(config.GlobalConfig.L1, config.GlobalConfig.L2, nodes)
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

	o, sharedKey, err := pi_t.FORMONION(c.PublicKey, c.PrivateKey, msgString, mixersAddr, gatekeepersAddr, destination.Address, publicKeys, metadata, config.GlobalConfig.D)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create onion")
	}

	onions = append(onions, queuedOnion{
		onion:     o[0][0],
		to:        mixersAddr[0],
		sharedKey: sharedKey,
	})

	c.status.AddSent(destination, routingPath, msg)

	//go func() {
	//	if err = api_functions.SendOnion(mixersAddr[0], c.Address, o[0][0]); err != nil {
	//		slog.Error("failed to send onions", err)
	//	}
	//}()

	//if checkpointOnions, err := c.createCheckpointOnions(routingPath, checkpoints, nodes, start); err != nil {
	//	return nil, err
	//} else {
	//	onions = append(onions, checkpointOnions...)
	//}

	return onions, nil
}

// createCheckpointOnions creates checkpoint onions for the routing path
func (c *Client) processCheckpoint(checkpoint structs.Checkpoint, nodes, clients []structs.PublicNodeApi) (onions []queuedOnion, err error) {
	onions = make([]queuedOnion, 0)

	mixers, gatekeepers, receiver, err := DetermineCheckpointRoutingPath(checkpoint, nodes, clients)
	if err != nil {
		return nil, pl.WrapError(err, "failed to determine checkpoint routing path")
	}

	mixersAddr := utils.Map(mixers, func(node structs.PublicNodeApi) string {
		return node.Address
	})

	gatekeepersAddr := utils.Map(gatekeepers, func(node structs.PublicNodeApi) string {
		return node.Address
	})

	path := append(append(mixers, gatekeepers...), receiver)

	checkpointPublicKeys := utils.Map(path, func(node structs.PublicNodeApi) string {
		return node.PublicKey
	})

	dummyMsg := structs.Message{
		From: c.Address,
		To:   utils.GetLast(path).Address,
		Msg:  "checkpoint onion",
		Hash: utils.GenerateUniqueHash(),
	}
	dummyPayload, err := json.Marshal(dummyMsg)
	if err != nil {
		return nil, pl.WrapError(err, "failed to marshal dummy message")
	}
	mString := base64.StdEncoding.EncodeToString(dummyPayload)

	metadata := make([]onion_model.Metadata, len(path)+1)
	for i := range metadata {
		if i == checkpoint.Layer {
			metadata[i] = onion_model.Metadata{
				Nonce: checkpoint.Nonce,
			}
		} else {
			metadata[i] = onion_model.Metadata{
				Nonce: "",
			}
		}
	}
	metadata = utils.InsertAtIndex(metadata, 0, onion_model.Metadata{})

	o, sharedKey, err := pi_t.FORMONION(c.PublicKey, c.PrivateKey, mString, mixersAddr, gatekeepersAddr, receiver.Address, checkpointPublicKeys, metadata, config.GlobalConfig.D)
	if err != nil {
		return nil, pl.WrapError(err, "failed to create checkpoint onion")
	}

	onions = append(onions, queuedOnion{
		onion:     o[0][0],
		to:        mixersAddr[0],
		sharedKey: sharedKey,
	})

	c.status.AddSent(utils.GetLast(path), path, dummyMsg)
	return onions, nil
}

func (c *Client) startRun(start structs.ClientStartRunApi) error {

	slog.Info("Client starting run", "num clients", len(start.Clients), "num mixers", len(start.Mixers), "num gatekeepers", len(start.Gatekeepers))
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(start.Mixers) == 0 {
		return pl.NewError("%s: no mixers", pl.GetFuncName())
	}
	if len(start.Gatekeepers) == 0 {
		return pl.NewError("%s: no gatekeepers", pl.GetFuncName())
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
			onion := onion
			go func() {
				defer wg.Done()
				if err = api_functions.SendOnion(onion.to, c.Address, onion.onion, onion.sharedKey); err != nil {
					slog.Error("failed to send onions", err)
				}
			}()
		}

		wg.Wait()

		c.Messages = make([]structs.Message, 0)
		return nil
	}
}

func (c *Client) Receive(oApi structs.OnionApi) error {
	sharedKey, err := keys.DecodeSharedKey(oApi.SharedKey)
	if err != nil {
		return pl.WrapError(err, "node.Receive(): failed to decode shared key")
	}
	if _, _, peeled, _, err := pi_t.PeelOnion(oApi.Onion, sharedKey); err != nil {
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
