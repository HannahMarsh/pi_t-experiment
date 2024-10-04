package main

import (
	_ "github.com/lib/pq"
)

func main() {

}

//
//func main() {
//	logLevel := flag.String("log-level", "debug", "Log level")
//
//	flag.Usage = func() {
//		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
//			slog.Error("Usage of %s:\n", err, os.Args[0])
//		}
//		flag.PrintDefaults()
//	}
//
//	flag.Parse()
//
//	pl.SetUpLogrusAndSlog(*logLevel)
//
//	// set GOMAXPROCS
//	if _, err := maxprocs.Set(); err != nil {
//		slog.Error("failed set max procs", err)
//		os.Exit(1)
//	}
//
//	if err, _ := config.InitGlobal(); err != nil {
//		slog.Error("failed to init config", err)
//		os.Exit(1)
//	}
//
//	slog.Info("⚡ init visualizer", "address", "http://localhost:8200")
//
//	time.Sleep(1 * time.Second)
//	http.HandleFunc("/data", serveData)
//	http.Handle("/", http.FileServer(http.Dir("./static")))
//	//http.Handle("/client", http.FileServer(http.Dir("./static/client")))
//	//http.Handle("/nodes", http.FileServer(http.Dir("./static/nodes")))
//	//http.Handle("/nodes/rounds", http.FileServer(http.Dir("./static/nodes/rounds")))
//
//	go func() {
//		if err := http.ListenAndServe(fmt.Sprintf(":%d", 8200), nil); err != nil {
//			if errors.Is(err, http.ErrServerClosed) {
//				slog.Info("HTTP server closed")
//			} else {
//				slog.Error("failed to start HTTP server", err)
//			}
//		}
//	}()
//
//	slog.Info("🌏 start visualizer...", "address", "http://localhost:8200")
//
//	quit := make(chan os.Signal, 1)
//	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
//
//	select {
//	case v := <-quit:
//		config.GlobalCancel()
//		slog.Info("", "signal.Notify", v)
//	case done := <-config.GlobalCtx.Done():
//		slog.Info("", "ctx.Done", done)
//	}
//
//}
//
//type Data struct {
//	Clients  map[string]structs.ClientStatus
//	Messages []Message
//	Nodes    map[string]structs.NodeStatus
//	mu       sync.RWMutex
//}
//
//type Message struct {
//	From         string
//	To           string
//	RoutingPath  []structs.PublicNodeApi
//	Msg          string
//	TimeSent     string
//	TimeReceived string
//	Hash         string
//}
//
//var (
//	data Data = Data{
//		Clients:  make(map[string]structs.ClientStatus),
//		Messages: make([]Message, 0),
//		Nodes:    make(map[string]structs.NodeStatus),
//	}
//)
//
//func serveData(w http.ResponseWriter, r *http.Request) {
//	// Set the response header to application/json
//	w.Header().Set("Content-Type", "application/json")
//
//	data.mu.Lock()
//	defer data.mu.Unlock()
//
//	for _, client := range config.GlobalConfig.Clients {
//		addr := fmt.Sprintf("http://%s:%d/status", client.Host, client.Port)
//		resp, err := http.Get(addr)
//		if err != nil {
//			slog.Error("failed to get client status", err)
//		} else {
//			defer resp.Body.Close()
//			var status structs.ClientStatus
//			if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
//				slog.Error("failed to decode client status", err)
//			} else {
//				data.Clients[addr] = status
//			}
//		}
//	}
//
//	for _, relay := range config.GlobalConfig.Relays {
//		addr := fmt.Sprintf("http://%s:%d/status", relay.Host, relay.Port)
//		resp, err := http.Get(addr)
//		if err != nil {
//			slog.Error("failed to get client status", err)
//		} else {
//			defer resp.Body.Close()
//			var status structs.NodeStatus
//			if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
//				slog.Error("failed to decode client status", err)
//			} else {
//				data.Nodes[addr] = status
//			}
//		}
//	}
//
//	m := make(map[string]Message)
//
//	for _, client := range data.Clients {
//		for _, sent := range client.MessagesSent {
//			mstr := sent.Message.Hash
//			if _, present := m[mstr]; present {
//				if m[mstr].From != sent.Message.From {
//					pl.LogNewError("from (%s) and sent.from (%s) do not match", m[mstr].From, sent.Message.From)
//				}
//				if m[mstr].To != sent.Message.To {
//					pl.LogNewError("to (%s) and sent.to (%s) do not match", m[mstr].To, sent.Message.To)
//				}
//				if m[mstr].Msg != sent.Message.Msg {
//					pl.LogNewError("msg (%s) and sent.msg (%s) do not match", m[mstr].Msg, sent.Message.Msg)
//				}
//				if m[mstr].Hash != sent.Message.Hash {
//					pl.LogNewError("hash (%s) and sent.hash (%s) do not match", m[mstr].Hash, sent.Message.Hash)
//				}
//				msg := Message{
//					From:         sent.Message.From,
//					To:           sent.Message.To,
//					RoutingPath:  sent.RoutingPath,
//					Msg:          sent.Message.Msg,
//					TimeSent:     sent.TimeSent.Format("2006-01-02 15:04:05"),
//					TimeReceived: m[mstr].TimeReceived,
//					Hash:         sent.Message.Hash,
//				}
//				m[mstr] = msg
//			} else {
//				m[mstr] = Message{
//					From:         sent.Message.From,
//					To:           sent.Message.To,
//					RoutingPath:  sent.RoutingPath,
//					Msg:          sent.Message.Msg,
//					TimeSent:     sent.TimeSent.Format("2006-01-02 15:04:05"),
//					TimeReceived: "not received",
//					Hash:         sent.Message.Hash,
//				}
//			}
//		}
//		for _, received := range client.MessagesReceived {
//			mstr := received.Message.Hash
//			if _, present := m[mstr]; present {
//				if m[mstr].From != received.Message.From {
//					pl.LogNewError("from (%s) and received.from (%s) do not match", m[mstr].From, received.Message.From)
//				}
//				if m[mstr].To != received.Message.To {
//					pl.LogNewError("to (%s) and received.to (%s) do not match", m[mstr].To, received.Message.To)
//				}
//				if m[mstr].Msg != received.Message.Msg {
//					pl.LogNewError("msg (%s) and received.msg (%s) do not match", m[mstr].Msg, received.Message.Msg)
//				}
//				if m[mstr].Hash != received.Message.Hash {
//					pl.LogNewError("hash (%s) and received.hash (%s) do not match", m[mstr].Hash, received.Message.Hash)
//				}
//				msg := Message{
//					From:         received.Message.From,
//					To:           received.Message.To,
//					RoutingPath:  m[mstr].RoutingPath,
//					Msg:          received.Message.Msg,
//					TimeSent:     m[mstr].TimeSent,
//					TimeReceived: received.TimeReceived.Format("2006-01-02 15:04:05"),
//					Hash:         received.Message.Hash,
//				}
//				m[mstr] = msg
//			} else {
//				m[mstr] = Message{
//					From:         received.Message.From,
//					To:           received.Message.To,
//					RoutingPath:  make([]structs.PublicNodeApi, 0),
//					Msg:          received.Message.Msg,
//					TimeSent:     "not sent",
//					TimeReceived: received.TimeReceived.Format("2006-01-02 15:04:05"),
//					Hash:         received.Message.Hash,
//				}
//			}
//		}
//	}
//
//	data.Messages = make([]Message, 0)
//	for _, msg := range m {
//		data.Messages = append(data.Messages, msg)
//	}
//
//	// Encode the data as JSON and write to the response
//	str, err := json.Marshal(data)
//	if err != nil {
//		slog.Error("failed to marshal data", err)
//	} else {
//		if _, err = w.Write(str); err != nil {
//			slog.Error("failed to write data", err)
//		}
//	}
//}
