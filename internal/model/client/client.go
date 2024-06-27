package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/api_functions"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/keys"
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
func DetermineRoutingPath(pathLength int, participants []structs.PublicNodeApi) ([]structs.PublicNodeApi, error) {
	if len(participants) < pathLength {
		return nil, pl.NewError("not enough participants to form routing path")
	}

	selectedNodes := make([]structs.PublicNodeApi, pathLength)
	perm := rand.Perm(len(participants))

	for i := 0; i < pathLength; i++ {
		selectedNodes[i] = participants[perm[i]]
	}

	adjustPathNodes(selectedNodes)
	return selectedNodes, nil
}

// adjustPathNodes adjusts the routing path to ensure the first node is a Mixer and the last is a gatekeeper
func adjustPathNodes(selectedNodes []structs.PublicNodeApi) {
	for i, node := range selectedNodes {
		if node.IsMixer {
			if i != 0 {
				utils.Swap(selectedNodes, i, 0)
			}
			break
		}
	}
	for i, node := range selectedNodes {
		if !node.IsMixer {
			if i != len(selectedNodes)-1 {
				utils.Swap(selectedNodes, i, len(selectedNodes)-1)
			}
			break
		}
	}
}

// DetermineCheckpointRoutingPath determines a routing path with a checkpoint
func DetermineCheckpointRoutingPath(pathLength int, nodes []structs.PublicNodeApi, participatingClients []structs.PublicNodeApi,
	checkpointReceiver structs.PublicNodeApi, round int) ([]structs.PublicNodeApi, error) {

	path, err := DetermineRoutingPath(pathLength-1, utils.Remove(nodes, func(p structs.PublicNodeApi) bool {
		return p.Address == checkpointReceiver.Address
	}))
	if err != nil {
		return nil, pl.WrapError(err, "failed to determine routing path")
	}
	return append(utils.InsertAtIndex(path, round, checkpointReceiver), utils.RandomElement(participatingClients)), nil
}

// formOnions forms the onions for the messages to be sent
func (c *Client) formOnions(start structs.StartRunApi) (map[string][]queuedOnion, error) {
	onions := make(map[string][]queuedOnion)

	nodes := utils.Filter(append(utils.Copy(start.Mixers), utils.Copy(start.Gatekeepers)...), func(node structs.PublicNodeApi) bool {
		return node.Address != c.Address && node.Address != ""
	})

	for _, msg := range c.Messages {
		if destination, found := utils.Find(start.ParticipatingClients, structs.PublicNodeApi{}, func(client structs.PublicNodeApi) bool {
			return client.Address == msg.To
		}); found {
			if err := c.processMessage(onions, msg, destination, nodes, start); err != nil {
				return nil, err
			}
		}
	}

	return onions, nil
}

type queuedOnion struct {
	to    string
	onion string
}

// processMessage processes a single message to form its onion
func (c *Client) processMessage(onions map[string][]queuedOnion, msg structs.Message, destination structs.PublicNodeApi, nodes []structs.PublicNodeApi, start structs.StartRunApi) error {
	msgString, err := json.Marshal(msg)
	if err != nil {
		return pl.WrapError(err, "failed to marshal message")
	}

	routingPath, err := DetermineRoutingPath(config.GlobalConfig.Rounds, nodes)
	if err != nil {
		return pl.WrapError(err, "failed to determine routing path")
	}

	routingPath = append(routingPath, destination)
	publicKeys := utils.Map(routingPath, func(node structs.PublicNodeApi) string {
		return node.PublicKey
	})
	addresses := utils.Map(routingPath, func(node structs.PublicNodeApi) string {
		return node.Address
	})

	addr, onion, checkpoints, err := pi_t.FormOnion(c.PrivateKey, c.PublicKey, msgString, publicKeys, addresses, -1, c.Address)
	if err != nil {
		return pl.WrapError(err, "failed to create onion")
	}

	if err := c.createCheckpointOnions(onions, routingPath, checkpoints, nodes, start); err != nil {
		return err
	}

	if _, present := onions[addr]; !present {
		onions[addr] = make([]queuedOnion, 0)
	}
	onions[addr] = append(onions[addr], queuedOnion{
		onion: onion,
		to:    addr,
	})
	c.status.AddSent(destination, routingPath, msg)

	return nil
}

// createCheckpointOnions creates checkpoint onions for the routing path
func (c *Client) createCheckpointOnions(onions map[string][]queuedOnion, routingPath []structs.PublicNodeApi, checkpoints []bool, nodes []structs.PublicNodeApi, start structs.StartRunApi) error {
	for i, node := range routingPath {
		if checkpoints[i] {
			path, err := DetermineCheckpointRoutingPath(config.GlobalConfig.Rounds, nodes, utils.Filter(start.ParticipatingClients, func(publicNodeApi structs.PublicNodeApi) bool {
				return publicNodeApi.Address != c.Address && publicNodeApi.Address != ""
			}), node, i)
			if err != nil {
				return pl.WrapError(err, "failed to determine checkpoint routing path")
			}

			checkpointPublicKeys := utils.Map(path, func(node structs.PublicNodeApi) string {
				return node.PublicKey
			})
			checkpointAddresses := utils.Map(path, func(node structs.PublicNodeApi) string {
				return node.Address
			})

			dummyMsg := structs.Message{
				From: c.Address,
				To:   utils.GetLast(path).Address,
				Msg:  "checkpoint onion",
				Hash: utils.GenerateUniqueHash(),
			}
			dummyPayload, err := json.Marshal(dummyMsg)
			if err != nil {
				return pl.WrapError(err, "failed to marshal dummy message")
			}

			firstHop, checkpointOnion, _, err := pi_t.FormOnion(c.PrivateKey, c.PublicKey, dummyPayload, checkpointPublicKeys, checkpointAddresses, i, c.Address)
			if err != nil {
				return pl.WrapError(err, "failed to create checkpoint onion")
			}

			if _, present := onions[firstHop]; !present {
				onions[firstHop] = make([]queuedOnion, 0)
			}
			onions[firstHop] = append(onions[firstHop], queuedOnion{
				onion: checkpointOnion,
				to:    firstHop,
			})
			c.status.AddSent(utils.GetLast(path), routingPath, dummyMsg)
		}
	}
	return nil
}

func (c *Client) startRun(start structs.StartRunApi) error {

	slog.Info("Client starting run", "num clients", len(start.ParticipatingClients), "num mixers", len(start.Mixers), "num gatekeepers", len(start.Gatekeepers))
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(start.Mixers) == 0 {
		return pl.NewError("%s: no mixers", pl.GetFuncName())
	}
	if len(start.Gatekeepers) == 0 {
		return pl.NewError("%s: no gatekeepers", pl.GetFuncName())
	}
	if len(start.ParticipatingClients) == 0 {
		return pl.NewError("%s: no participating clients", pl.GetFuncName())
	}

	if !utils.Contains(start.ParticipatingClients, func(client structs.PublicNodeApi) bool {
		return client.ID == c.ID
	}) {
		return nil
	}

	if toSend, err := c.formOnions(start); err != nil {
		return pl.WrapError(err, "failed to form toSend")
	} else {
		numToSend := 0

		for _, onions := range toSend {
			numToSend += len(onions)
		}
		slog.Info("Client sending onions", "num_onions", numToSend)

		var wg sync.WaitGroup
		wg.Add(numToSend)
		for addr, onions := range toSend {
			for _, onion := range onions {
				onion := onion
				go func() {
					defer wg.Done()
					if err = api_functions.SendOnion(addr, c.Address, onion.onion); err != nil {
						slog.Error("failed to send onions", err)
					}
				}()
			}
		}

		wg.Wait()

		c.Messages = make([]structs.Message, 0)
		return nil
	}
}

//
//func (c *Client) sendOnion(addr string, onion api.OnionApi) error {
//	slog.Info("Client sending onion", "from", onion.From, "to", onion.To)
//	url := fmt.Sprintf("%s/receive", addr)
//
//	if data, err2 := json.Marshal(onion); err2 != nil {
//		slog.Error("failed to marshal msgs", err2)
//	} else if resp, err3 := http.Post(url, "application/json", bytes.NewBuffer(data)); err3 != nil {
//		return pl.WrapError(err3, "failed to send POST request with onion to first mixer")
//	} else {
//		defer func(Body io.ReadCloser) {
//			if err4 := Body.Close(); err4 != nil {
//				slog.Error(pl.GetFuncName()+": Error closing response body", err4)
//			}
//		}(resp.Body)
//		if resp.StatusCode != http.StatusOK {
//			return pl.NewError("%s: Failed to send to first node(url=%s), status code: %d, status: %s", pl.GetFuncName(), url, resp.StatusCode, resp.Status)
//		} else {
//			slog.Info("Client sent onion to first mixer", "mixer_address", addr)
//		}
//	}
//	return nil
//}

func (c *Client) Receive(o string) error {
	if peeled, bruises, _, _, err := pi_t.PeelOnion(o, c.PrivateKey); err != nil {
		return pl.WrapError(err, "node.Receive(): failed to remove layer")
	} else {
		slog.Info("Client received onion", "bruises", bruises, "from", peeled.LastHop, "to", peeled.NextHop, "layer", peeled.Layer, "is_checkpoint_onion", peeled.IsCheckpointOnion)
		if peeled.NextHop == "" {
			var msg structs.Message
			if err2 := json.Unmarshal([]byte(peeled.Payload), &msg); err2 != nil {
				return pl.WrapError(err2, "node.Receive(): failed to unmarshal message")
			}
			slog.Info("Received message", "from", msg.From, "to", msg.To, "msg", msg.Msg)

			c.status.AddReceived(msg)

		} else {
			return pl.NewError("Received onion", "destination", peeled.NextHop)
		}
	}
	return nil
}

func (c *Client) GetStatus() string {
	return c.status.GetStatus()
}

func (c *Client) RegisterIntentToSend(messages []structs.Message) error {

	//	slog.Info("Client registering intent to send", "id", c.ID, "num_messages", len(messages))

	to := utils.Map(messages, func(m structs.Message) structs.PublicNodeApi {
		if f, found := utils.Find(c.OtherClients, structs.PublicNodeApi{}, func(c structs.PublicNodeApi) bool {
			return c.Address == m.To
		}); found {
			return f
		} else {
			return f
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
