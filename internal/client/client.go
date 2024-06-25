package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
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
	Adddress         string
	PrivateKey       string
	PublicKey        string
	SessionKeys      map[string][]byte
	ActiveNodes      []api.PublicNodeApi
	OtherClients     []api.PublicNodeApi
	Messages         []api.Message
	mu               sync.RWMutex
	BulletinBoardUrl string
	status           *api.ClientStatus
}

// NewNode creates a new node
func NewClient(id int, host string, port int, bulletinBoardUrl string) (*Client, error) {
	if privateKey, publicKey, err := pi_t.KeyGen(); err != nil {
		return nil, pl.WrapError(err, "node.NewClient(): failed to generate key pair")
	} else {
		c := &Client{
			ID:               id,
			Host:             host,
			Port:             port,
			Adddress:         fmt.Sprintf("http://%s:%d", host, port),
			PublicKey:        publicKey,
			PrivateKey:       privateKey,
			SessionKeys:      make(map[string][]byte),
			ActiveNodes:      make([]api.PublicNodeApi, 0),
			BulletinBoardUrl: bulletinBoardUrl,
			Messages:         make([]api.Message, 0),
			OtherClients:     make([]api.PublicNodeApi, 0),
			status: &api.ClientStatus{
				MessagesSent:     make([]api.Sent, 0),
				MessagesReceived: make([]api.Received, 0),
			},
		}

		if err2 := c.RegisterWithBulletinBoard(); err2 != nil {
			return nil, pl.WrapError(err2, "%s: failed to register with bulletin board", pl.GetFuncName(id, host, port, bulletinBoardUrl))
		}

		return c, nil
	}
}

func (c *Client) RegisterWithBulletinBoard() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if data, err := json.Marshal(api.PublicNodeApi{
		ID:        c.ID,
		Address:   c.Adddress,
		PublicKey: c.PublicKey,
	}); err != nil {
		return pl.WrapError(err, "Client.UpdateBulletinBoard(): failed to marshal Client info")
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

func (c *Client) StartGeneratingMessages(client_addresses []string) {
	slog.Info("Client starting to generate messages", "id", c.ID)
	var msgNum int = 0
	for {
		select {
		case <-config.GlobalCtx.Done():
			slog.Info(pl.GetFuncName()+": ctx.done -> Client stopping to generate messages", "id", c.ID)
			return // Exit if context is cancelled
		default:
			messages := make([]api.Message, 0)
			for _, addr := range client_addresses {
				if addr != c.Adddress {
					messages = append(messages, api.Message{
						From: c.Adddress,
						To:   addr,
						Msg:  fmt.Sprintf("Msg#%d from client(id=%d)", msgNum, c.ID),
					})
					msgNum++
				}
			}
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				c.mu.Lock()
				defer func() {
					c.mu.Unlock()
					wg.Done()
				}()
				if err := c.RegisterIntentToSend(messages); err != nil {
					slog.Error(pl.GetFuncName()+": Error registering intent to send", err)
				} else {
					c.Messages = append(c.Messages, messages...)
				}
			}()
			wg.Wait()
		}
		time.Sleep(5 * time.Second)
	}
}

var rand = rng.New(rng.NewSource(time.Now().UnixNano()))

// DetermineRoutingPath determines a random routing path of a given length
func DetermineRoutingPath(pathLength int, participants []api.PublicNodeApi) ([]api.PublicNodeApi, error) {
	if len(participants) < pathLength {
		return nil, pl.NewError("%s: not enough participants to form routing path", pl.GetFuncName(pathLength))
	}

	selectedNodes := make([]api.PublicNodeApi, pathLength)
	perm := rand.Perm(len(participants))

	for i := 0; i < pathLength; i++ {
		selectedNodes[i] = participants[perm[i]]
	}

	// swap first node with a mixer if it is a gatekeeper
	for i, node := range selectedNodes {
		if node.IsMixer {
			if i == 0 {
				break
			}
			temp := selectedNodes[i]
			selectedNodes[i] = selectedNodes[0]
			selectedNodes[0] = temp
			break
		}
	}

	// swap last node with a gatekeeper if it is a mixer
	for i, node := range selectedNodes {
		if !node.IsMixer {
			if i == len(selectedNodes)-1 {
				break
			}
			temp := selectedNodes[i]
			selectedNodes[i] = selectedNodes[len(selectedNodes)-1]
			selectedNodes[len(selectedNodes)-1] = temp
			break
		}
	}

	return selectedNodes, nil
}

func (c *Client) formOnions(start api.StartRunApi) (map[string][]api.OnionApi, error) {

	onions := make(map[string][]api.OnionApi)

	nodes := utils.Copy(start.Mixers)
	nodes = append(nodes, utils.Copy(start.Gatekeepers)...)

	nodes = utils.Filter(nodes, func(node api.PublicNodeApi) bool {
		return node.Address != c.Adddress && node.Address != ""
	})

	numMessagesToSend := make(map[string]int)

	for _, msg := range c.Messages {
		if _, found := numMessagesToSend[msg.To]; !found {
			numMessagesToSend[msg.To] = 0
		}
		numMessagesToSend[msg.To]++
	}

	for addr, numMessages := range numMessagesToSend {
		if numMessages < start.NumMessagesPerClient {
			numDummyNeeded := start.NumMessagesPerClient - numMessages
			for i := 0; i < numDummyNeeded; i++ {
				c.Messages = append(c.Messages, api.Message{
					From: c.Adddress,
					To:   addr,
					Msg:  "dummy",
				})
			}
		}
	}

	for _, msg := range c.Messages {
		if destination, found := utils.Find(start.ParticipatingClients, api.PublicNodeApi{}, func(client api.PublicNodeApi) bool {
			return client.Address == msg.To
		}); found {

			//slog.Info("Client forming onion", "from", c.Adddress, "to", destination.Address, "msg", msg.Msg)

			if msgString, err := json.Marshal(msg); err != nil {
				return nil, pl.WrapError(err, "failed to marshal message")
			} else if routingPath, err2 := DetermineRoutingPath(config.GlobalConfig.Rounds, nodes); err2 != nil {
				return nil, pl.WrapError(err2, "failed to determine routing path")
			} else {
				routingPath = append(routingPath, destination)
				publicKeys := utils.Map(routingPath, func(node api.PublicNodeApi) string {
					return node.PublicKey
				})
				addresses := utils.Map(routingPath, func(node api.PublicNodeApi) string {
					return node.Address
				})
				slog.Info("routing path", "path", addresses)
				if addr, onion, err3 := pi_t.FormOnion(msgString, publicKeys, addresses, -1); err3 != nil {
					return nil, pl.WrapError(err3, "failed to create onion")
				} else {
					if _, present := onions[addr]; !present {
						onions[addr] = make([]api.OnionApi, 0)
					}
					onions[addr] = append(onions[msg.To], api.OnionApi{
						Onion: onion,
						From:  c.Adddress,
						To:    addr,
					})
					c.status.AddSent(destination, routingPath, msg)
				}
			}
		}
	}

	return onions, nil
}

func (c *Client) startRun(start api.StartRunApi) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(start.Mixers) == 0 {
		return false, pl.NewError("%s: no active nodes", pl.GetFuncName())
	}
	if len(start.ParticipatingClients) == 0 {
		return false, pl.NewError("%s: no participating clients", pl.GetFuncName())
	}

	doParticipate := false
	for _, client := range start.ParticipatingClients {
		if client.ID == c.ID {
			doParticipate = true
		}
	}

	if !doParticipate {
		return false, nil
	}

	if toSend, err := c.formOnions(start); err != nil {
		return true, pl.WrapError(err, "failed to form toSend")
	} else {
		for addr, onions := range toSend {
			for _, onion := range onions {
				url := fmt.Sprintf("%s/receive", addr)

				if data, err2 := json.Marshal(onion); err2 != nil {
					slog.Error("failed to marshal msgs", err2)
				} else if resp, err3 := http.Post(url, "application/json", bytes.NewBuffer(data)); err3 != nil {
					return true, pl.WrapError(err3, "failed to send POST request with onion to first mixer")
				} else {
					defer func(Body io.ReadCloser) {
						if err4 := Body.Close(); err4 != nil {
							slog.Error(pl.GetFuncName()+": Error closing response body", err4)
						}
					}(resp.Body)
					if resp.StatusCode != http.StatusOK {
						return true, pl.NewError("%s: Failed to send to first node(url=%s), status code: %d, status: %s", pl.GetFuncName(), url, resp.StatusCode, resp.Status)
					} else {
						slog.Info("Client sent onion to first mixer", "mixer_address", addr)
					}
				}
			}
		}
	}
	c.Messages = make([]api.Message, 0)
	return true, nil
}

//func (c *Client) RegisterNode(nodeID string, nodePubKey *ecdh.PublicKey) error {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	if sharedKey, err := utils.ComputeSharedKey(c.PrivateKey, nodePubKey); err != nil {
//		return pl.WrapError(err, "error computing shared key")
//	} else {
//		c.SessionKeys[nodeID] = sharedKey
//		return nil
//	}
//}

func (c *Client) Receive(o string) error {
	if peeled, err := pi_t.PeelOnion(o, c.PrivateKey); err != nil {
		return pl.WrapError(err, "node.Receive(): failed to remove layer")
	} else {
		if peeled.NextHop == "" {
			var msg api.Message
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

func (c *Client) RegisterIntentToSend(messages []api.Message) error {

	//	slog.Info("Client registering intent to send", "id", c.ID, "num_messages", len(messages))

	to := utils.Map(messages, func(m api.Message) api.PublicNodeApi {
		if f, found := utils.Find(c.OtherClients, api.PublicNodeApi{}, func(c api.PublicNodeApi) bool {
			return c.Address == m.To
		}); found {
			return f
		} else {
			return f
		}
	})
	if data, err := json.Marshal(api.IntentToSend{
		From: api.PublicNodeApi{
			ID:        c.ID,
			Address:   c.Adddress,
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
