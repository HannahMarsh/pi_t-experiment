package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/metrics"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/client"
	"go.uber.org/automaxprocs/maxprocs"
	"log/slog"
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

	// Set up logrus with the specified log level.
	pl.SetUpLogrusAndSlog(*logLevel)

	// Automatically adjust the GOMAXPROCS setting based on the number of available CPU cores.
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	// Initialize global configurations by loading them from config/config.yml
	if err, _ := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	cfg := config.GlobalConfig

	// Find the client configuration that matches the provided ID.
	var clientConfig *config.Client
	for _, c := range cfg.Clients {
		if c.ID == *id {
			clientConfig = &c
			break
		}
	}

	if clientConfig == nil {
		pl.LogNewError("Invalid id. Failed to get newClient config for id=%d", *id)
		os.Exit(1)
	}

	slog.Info("⚡ init newClient", "id", *id)

	// Construct the full URL for the Bulletin Board
	bulletinBoardAddress := fmt.Sprintf("http://%s:%d", cfg.BulletinBoard.Host, cfg.BulletinBoard.Port)

	var newClient *client.Client
	// Attempt to create a new client instance, retrying every 5 seconds upon failure (in case the bulletin board isn't ready yet).
	for {
		if n, err := client.NewClient(clientConfig.ID, clientConfig.Host, clientConfig.Port, bulletinBoardAddress); err != nil {
			slog.Error("failed to create new c. Trying again in 5 seconds. ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			// If successful, assign the new client instance and break out of the loop.
			newClient = n
			break
		}
	}

	// Create a channel to receive OS signals (like SIGINT or SIGTERM) to handle graceful shutdowns.
	quit := make(chan os.Signal, 1)
	// Notify the quit channel when specific OS signals are received.
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Set up HTTP handlers
	http.HandleFunc("/receive", newClient.HandleReceive)
	http.HandleFunc("/start", newClient.HandleStartRun)
	http.HandleFunc("/status", newClient.HandleGetStatus)
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Shutdown signal received")
		quit <- os.Signal(syscall.SIGTERM) // signal shutdown
		_, err := w.Write([]byte("Shutting down..."))
		if err != nil {
			slog.Error("Error", err)
		}
	})

	// Serve Prometheus metrics in a separate goroutine.
	shutdownMetrics := metrics.ServeMetrics(clientConfig.PrometheusPort, metrics.MSG_SENT, metrics.MSG_RECEIVED, metrics.ONION_SIZE)

	// Start the HTTP server
	go func() {
		// Construct the address for the HTTP server based on the client's port.
		address := fmt.Sprintf(":%d", clientConfig.Port)
		// Attempt to start the HTTP server.
		if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	slog.Info("🌏 start newClient...", "address", fmt.Sprintf("http://%s:%d", clientConfig.Host, clientConfig.Port))

	// Start generating messages in a separate goroutine.
	go newClient.StartGeneratingMessages()

	// Wait for either an OS signal to quit or the global context to be canceled
	select {
	case v := <-quit: // OS signal is received
		config.GlobalCancel()
		shutdownMetrics()
		slog.Info("", "signal.Notify", v)
	case done := <-config.GlobalCtx.Done(): // global context is canceled
		slog.Info("", "ctx.Done", done)
		shutdownMetrics()
	}

}
