package main

import (
	"errors"
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/metrics"
	"github.com/HannahMarsh/pi_t-experiment/internal/model/relay"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
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
	id_ := flag.Int("id", -1, "ID of the newClient (required)")
	ip_ := flag.String("host", "x", "IP address of the relay")
	port_ := flag.Int("port", 8080, "Port of the client")
	promPort_ := flag.Int("promPort", 8200, "Port of the relay's Prometheus metrics")
	logLevel_ := flag.String("log-level", "debug", "Log level")

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0]); err != nil {
			slog.Error("Usage of %s:\n", err, os.Args[0])
		}
		flag.PrintDefaults()
	}

	flag.Parse()

	id := *id_
	ip := *ip_
	port := *port_
	promPort := *promPort_
	logLevel := *logLevel_

	pl.SetUpLogrusAndSlog(logLevel)

	// Check if the required flag is provided
	if id == -1 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: the -id flag is required\n")
		flag.Usage()
		os.Exit(2)
	}

	if ip == "x" {
		IP, err := utils.GetPublicIP()
		if err != nil {
			slog.Error("failed to get public IP", err)
			os.Exit(1)
		}
		slog.Info("", "IP", IP.IP, "Hostname", IP.HostName, "City", IP.City, "Region", IP.Region, "Country", IP.Country, "Location", IP.Location, "Org", IP.Org, "Postal", IP.Postal, "Timezone", IP.Timezone, "ReadMe", IP.ReadMe)
		ip = IP.IP
	}

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

	slog.Info("⚡ init newRelay", "id", id)

	var newRelay *relay.Relay
	// Attempt to create a new relay instance, retrying every 5 seconds upon failure (in case the bulletin board isn't ready yet).
	for {
		if n, err := relay.NewRelay(id, ip, port, promPort, config.GetBulletinBoardAddress()); err != nil {
			slog.Error("failed to create newRelay. Trying again in 5 seconds. ", err)
			time.Sleep(5 * time.Second)
			continue
		} else {
			// If successful, assign the new relay instance and break out of the loop.
			newRelay = n
			break
		}
	}

	// Create a channel to receive OS signals (like SIGINT or SIGTERM) to handle graceful shutdowns.
	quit := make(chan os.Signal, 1)
	// Notify the quit channel when specific OS signals are received.
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Set up HTTP handlers
	http.HandleFunc("/receive", newRelay.HandleReceiveOnion)
	http.HandleFunc("/start", newRelay.HandleStartRun)
	http.HandleFunc("/status", newRelay.HandleGetStatus)
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Shutdown signal received")
		quit <- os.Signal(syscall.SIGTERM) // signal shutdown
		_, err := w.Write([]byte("Shutting down..."))
		if err != nil {
			slog.Error("Error", err)
		}
	})

	// Serve Prometheus metrics in a separate goroutine.
	shutdownMetrics := metrics.ServeMetrics(promPort, metrics.PROCESSING_TIME, metrics.ONION_COUNT, metrics.ONION_SIZE)

	// Start the HTTP server
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("HTTP server closed")
			} else {
				slog.Error("failed to start HTTP server", err)
			}
		}
	}()

	slog.Info("🌏 start newRelay...", "address", fmt.Sprintf("http://%s:%d", ip, port))

	// Wait for either an OS signal to quit or the global context to be canceled
	select {
	case v := <-quit: // OS signal is received
		config.GlobalCancel()
		shutdownMetrics()
		slog.Info("", "signal.Notify", v)
	case done := <-config.GlobalCtx.Done(): // global context is canceled
		slog.Info("", "ctx.Done", done)
	}

}
