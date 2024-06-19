package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/HannahMarsh/pi_t-experiment/cmd/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/node"
	"github.com/HannahMarsh/pi_t-experiment/pkg/infrastructure/logger"
	"github.com/sirupsen/logrus"
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
		if _, err := fmt.Fprintf(os.Stderr, "Error: the -id flag is required\n"); err != nil {
			slog.Error("Error: the -id flag is required\n", err)
		}
		flag.Usage()
		os.Exit(2)
	}

	// set GOMAXPROCS
	_, err := maxprocs.Set()
	if err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err = config.InitConfig(); err != nil {
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

	// set up logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logger.ConvertLogLevel(*logLevel))

	// integrate Logrus with the slog logger
	slog.New(logger.NewLogrusHandler(logrus.StandardLogger()))

	baddress := fmt.Sprintf("http://%s:%d", cfg.BulletinBoard.Host, cfg.BulletinBoard.Port)

	var newNode *node.Node
	for {
		if newNode, err = node.NewNode(nodeConfig.ID, nodeConfig.Host, nodeConfig.Port, baddress); err != nil {
			slog.Error("failed to create newNode. Trying again in 5 seconds. ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			break
		}
	}

	http.HandleFunc("/receive", newNode.HandleReceive)
	http.HandleFunc("/requestMsg", newNode.HandleClientRequest)
	http.HandleFunc("/start", newNode.HandleStartRun)

	go func() {
		address := fmt.Sprintf(":%d", nodeConfig.Port)
		if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	slog.Info("🌏 start newNode...", "address", fmt.Sprintf("http://%s:%d", nodeConfig.Host, nodeConfig.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		cancel()
		slog.Info("signal.Notify", v)
	case done := <-ctx.Done():
		slog.Info("ctx.Done", done)
	}

}
