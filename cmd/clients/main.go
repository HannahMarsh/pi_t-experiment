package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/client"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Define command-line flags
	id := flag.Int("id", -1, "ID of the newClient (required)")
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check if the required flag is provided
	if *id == -1 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: the -id flag is required\n")
		flag.Usage()
		os.Exit(2)
	}

	pl.SetUpLogrusAndSlog(*logLevel)

	// set GOMAXPROCS
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	cfg := config.GlobalConfig

	var clientConfig *config.Client
	for _, client := range cfg.Clients {
		if client.ID == *id {
			clientConfig = &client
			break
		}
	}

	if clientConfig == nil {
		slog.Error("invalid id", errors.New(fmt.Sprintf("failed to get newClient config for id=%d", *id)))
		os.Exit(1)
	}

	slog.Info("⚡ init newClient", "heartbeat_interval", cfg.HeartbeatInterval, "id", *id)

	baddress := fmt.Sprintf("http://%s:%d", cfg.BulletinBoard.Host, cfg.BulletinBoard.Port)

	var newClient *client.Client
	for {
		if n, err := client.NewClient(clientConfig.ID, clientConfig.Host, clientConfig.Port, baddress); err != nil {
			slog.Error("failed to create new client. Trying again in 5 seconds. ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			newClient = n
			break
		}
	}

	http.HandleFunc("/receive", newClient.HandleReceive)
	http.HandleFunc("/start", newClient.HandleStartRun)
	http.HandleFunc("/status", newClient.HandleGetStatus)

	go func() {
		address := fmt.Sprintf(":%d", clientConfig.Port)
		if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	slog.Info("🌏 start newClient...", "address", fmt.Sprintf("http://%s:%d", clientConfig.Host, clientConfig.Port))

	clientAddresses := utils.Map(cfg.Clients, func(n config.Client) string {
		return fmt.Sprintf("http://%s:%d", n.Host, n.Port)
	})

	go newClient.StartGeneratingMessages(clientAddresses)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		config.GlobalCancel()
		slog.Info("signal.Notify", v)
	case done := <-config.GlobalCtx.Done():
		slog.Info("ctx.Done", done)
	}

}
