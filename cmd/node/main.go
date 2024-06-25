package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/node"
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
	id := flag.Int("id", -1, "ID of the newNode (required)")
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

	var nodeConfig *config.Node
	for _, n := range cfg.Nodes {
		if n.ID == *id {
			nodeConfig = &n
			break
		}
	}

	if nodeConfig == nil {
		slog.Error("invalid id", errors.New(fmt.Sprintf("failed to get newNode config for id=%d", *id)))
		os.Exit(1)
	}

	slog.Info("⚡ init newNode", "heartbeat_interval", cfg.HeartbeatInterval, "id", *id)

	baddress := fmt.Sprintf("http://%s:%d", cfg.BulletinBoard.Host, cfg.BulletinBoard.Port)

	var newNode *node.Node
	for {
		if n, err := node.NewNode(nodeConfig.ID, nodeConfig.Host, nodeConfig.Port, baddress, nodeConfig.IsMixer); err != nil {
			slog.Error("failed to create newNode. Trying again in 5 seconds. ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			newNode = n
			break
		}
	}

	http.HandleFunc("/receive", newNode.HandleReceiveOnion)
	http.HandleFunc("/start", newNode.HandleStartRun)
	http.HandleFunc("/status", newNode.HandleGetStatus)

	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", nodeConfig.Port), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 start newNode...", "address", fmt.Sprintf("http://%s:%d", nodeConfig.Host, nodeConfig.Port))

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
