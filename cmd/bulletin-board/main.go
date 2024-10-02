package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/bulletin_board"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/automaxprocs/maxprocs"
	"log/slog"
)

func main() {
	// Define command-line flags
	logLevel := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	// Set up logrus with the specified log level.
	pl.SetUpLogrusAndSlog(*logLevel)

	// Automatically adjust the GOMAXPROCS setting based on the number of available CPU cores.
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set maxprocs", err)
		os.Exit(1)
	}

	// Initialize global configurations by loading them from config/config.yml
	if err, _ := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	// Construct the full URL for the Bulletin Board
	url := fmt.Sprintf("https://%s:%d", config.GetBulletinBoardHost(), config.GetBulletinBoardPort())

	slog.Info("⚡ init Bulletin board", "url", url)

	// Create a new instance of the Bulletin Board with the current configuration.
	bulletinBoard := bulletin_board.NewBulletinBoard()

	// Start the Bulletin Board's main operations in a new goroutine
	go func() {
		err := bulletinBoard.StartProtocol()
		if err != nil {
			slog.Error("failed to start runs", err)
			config.GlobalCancel()
		}
	}()

	// Create a channel to receive OS signals (like SIGINT or SIGTERM) to handle graceful shutdowns.
	quit := make(chan os.Signal, 1)
	// Notify the quit channel when specific OS signals are received.
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Set up HTTP handlers
	http.HandleFunc("/registerRelay", bulletinBoard.HandleRegisterRelay)
	http.HandleFunc("/registerClient", bulletinBoard.HandleRegisterClient)
	//http.HandleFunc("/registerIntentToSend", bulletinBoard.HandleRegisterIntentToSend)
	http.HandleFunc("/updateRelay", bulletinBoard.HandleUpdateRelayInfo)
	http.HandleFunc("/updateConfig", bulletinBoard.HandleUpdateConfig)
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Shutdown signal received")
		quit <- os.Signal(syscall.SIGTERM) // signal shutdown
		_, err := w.Write([]byte("Shutting down..."))
		if err != nil {
			slog.Error("Error", err)
		}
	})

	// Start the HTTP server
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", config.GetBulletinBoardPort()), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) { // Check if the server was closed intentionally (normal shutdown).
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 starting bulletin board...", "address", url)

	// Wait for either an OS signal to quit or the global context to be canceled
	select {
	case v := <-quit: // OS signal is received
		config.GlobalCancel()
		bulletinBoard.Shutdown()
		slog.Info("", "signal.Notify", v)
	case done := <-config.GlobalCtx.Done(): // global context is canceled
		slog.Info("", "ctx.Done", done)
		bulletinBoard.Shutdown()
	}
}
