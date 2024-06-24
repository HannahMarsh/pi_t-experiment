package client

import (
	"bytes"
	"crypto/ecdh"
	"encoding/json"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api"
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
	Adddress         string
	PrivateKey       *ecdh.PrivateKey
	PublicKey        *ecdh.PublicKey
	SessionKeys      map[string][]byte
	ActiveNodes      []api.PublicNodeApi
	OtherClients     []api.PublicClientApi
	Messages         []api.Message
	mu               sync.RWMutex
	BulletinBoardUrl string
}

// NewNode creates a new node
func NewClient(id int, host string, port int, bulletinBoardUrl string) (*Client, error) {
	if privateKey, publicKey, err := utils.GenerateECDHKeyPair(); err != nil {
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
	if data, err := json.Marshal(api.PublicClientApi{
		ID:        c.ID,
		Address:   c.Adddress,
		PublicKey: c.PublicKey,
	}); err != nil {
		return pl.WrapError(err, "Client.UpdateBulletinBoard(): failed to marshal Client info")
	} else {
		url := c.BulletinBoardUrl + "registerClient"
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
			}
			return nil
		}
	}
}

func (c *Client) StartGeneratingMessages(client_addresses []string) {
	for {
		select {
		case <-config.GlobalCtx.Done():
			return // Exit if context is cancelled
		default:
			messages := make([]api.Message, 0)
			for _, addr := range client_addresses {
				messages = append(messages, api.Message{
					From: c.Adddress,
					To:   addr,
					Msg:  fmt.Sprintf("msg from client(id=%d)", c.ID),
				})
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

func (c *Client) formOnions(start api.StartRunApi) map[string][]api.OnionApi {

}

//func (c *Client) InitializeSessionKeys(activeNodes []api.PublicNodeApi) ([]api.StartSession, error) {
//	sharedKeys := make([]api.StartSession, 0)
//	for _, node := range activeNodes {
//		if sharedKey, err := utils.ComputeSharedKey(c.PrivateKey, node.PublicKey); err != nil {
//			return nil, pl.WrapError(err, "%s: error computing shared key", pl.GetFuncName())
//		} else {
//			sharedKeys = append(sharedKeys, api.StartSession{
//				Node: node,
//				Client: api.PublicClientApi{
//					ID:        c.ID,
//					Address:   c.Adddress,
//					PublicKey: c.PublicKey,
//				},
//				SessionKey: sharedKey,
//			})
//		}
//	}
//	return sharedKeys, nil
//}

func (c *Client) startRun(start api.StartRunApi) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(start.ActiveNodes) == 0 {
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

	//var sessions []api.StartSession
	//if sharedKeys, err := c.InitializeSessionKeys(start.ActiveNodes); err != nil {
	//	return true, pl.WrapError(err, "%s: error initializing session keys", pl.GetFuncName())
	//} else {
	//	for _, startSesssion := range sharedKeys {
	//		url := fmt.Sprintf("%s/receiveOnion", startSesssion.Node.Address)
	//
	//		if data, err2 := json.Marshal(startSesssion); err2 != nil {
	//			return true, pl.WrapError(err2, "%s: failed to marshal msgs", pl.GetFuncName())
	//		} else if resp, err3 := http.Post(url, "application/json", bytes.NewBuffer(data)); err3 != nil {
	//			return true, pl.WrapError(err3, "%s: failed to send POST request to node to share session key", pl.GetFuncName())
	//		} else {
	//			defer func(Body io.ReadCloser) {
	//				if err4 := Body.Close(); err4 != nil {
	//					slog.Error(pl.GetFuncName()+": Error closing response body", err4)
	//				}
	//			}(resp.Body)
	//			if resp.StatusCode != http.StatusOK {
	//				return true, pl.NewError("%s: Failed to send to first node, status code: %d, status: %s", pl.GetFuncName(), resp.StatusCode, resp.Status)
	//			} else {
	//				var s api.StartSession
	//				if err4 := json.NewDecoder(resp.Body).Decode(&s); err4 != nil {
	//					return true, pl.WrapError(err4, "%s: Error closing response body", pl.GetFuncName())
	//				} else {
	//					sessions = append(sessions, s)
	//				}
	//			}
	//		}
	//	}
	//
	//}

	onions := c.formOnions(start)

	for addr, onion := range onions {
		url := fmt.Sprintf("%s/receiveOnion", addr)

		if data, err := json.Marshal(onion); err != nil {
			slog.Error("failed to marshal msgs", err)
		} else if resp, err2 := http.Post(url, "application/json", bytes.NewBuffer(data)); err2 != nil {
			return true, pl.WrapError(err2, "failed to send POST request with onion to first mixer")
		} else {
			defer func(Body io.ReadCloser) {
				if err3 := Body.Close(); err3 != nil {
					slog.Error(pl.GetFuncName()+": Error closing response body", err3)
				}
			}(resp.Body)
			if resp.StatusCode != http.StatusOK {
				return true, pl.NewError("%s: Failed to send to first node, status code: %d, status: %s", pl.GetFuncName(), resp.StatusCode, resp.Status)
			}
		}
	}
	return true, nil
}

func (c *Client) RegisterNode(nodeID string, nodePubKey *ecdh.PublicKey) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if sharedKey, err := utils.ComputeSharedKey(c.PrivateKey, nodePubKey); err != nil {
		return pl.WrapError(err, "error computing shared key")
	} else {
		c.SessionKeys[nodeID] = sharedKey
		return nil
	}
}

func (c *Client) Receive(o string) error {
	return nil
}

func (c *Client) RegisterIntentToSend(messages []api.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	to := utils.Map(messages, func(m api.Message) api.PublicClientApi {
		return *utils.Find(c.OtherClients, func(c api.PublicClientApi) bool {
			return c.Address == m.To
		})
	})
	if data, err := json.Marshal(api.IntentToSend{
		From: api.PublicClientApi{
			ID:        c.ID,
			Address:   c.Adddress,
			PublicKey: c.PublicKey,
		},
		To: to,
	}); err != nil {
		return pl.WrapError(err, "%s: failed to marshal Client info", pl.GetFuncName())
	} else {
		url := c.BulletinBoardUrl + "registerIntentToSend"
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
			}
			return nil
		}
	}
}
