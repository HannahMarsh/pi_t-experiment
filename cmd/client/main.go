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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

var stopNTC func()
var newClient *client.Client

//var shutdownMetrics func()

func main() {
	// Define command-line flags
	id_ := flag.Int("id", -1, "ID of the newClient (required)")
	ip_ := flag.String("host", "x", "IP address of the client")
	port_ := flag.Int("port", 0, "Port of the client")
	promPort_ := flag.Int("promPort", 0, "Port of the client's Prometheus metrics")
	logLevel_ := flag.String("log-level", "info", "Log level")

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

	stopNTC = utils.StartNTP()

	if port == 0 {
		var err error
		port, err = utils.GetAvailablePort()
		if err != nil {
			slog.Error("failed to get available port", err)
			os.Exit(1)
		}
	}

	if promPort == 0 {
		var err error
		promPort, err = utils.GetAvailablePort()
		if err != nil {
			slog.Error("failed to get available port", err)
			os.Exit(1)
		}
	}

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

	slog.Info("⚡ init newClient", "id", id)

	// Attempt to create a new client instance, retrying every 5 seconds upon failure (in case the bulletin board isn't ready yet).
	for {
		if n, err := client.NewClient(id, ip, port, promPort, config.GetBulletinBoardAddress()); err != nil {
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
	http.HandleFunc("/register", newClient.HandleRegisterWithBulletinBoard)
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Shutdown signal received")
		quit <- os.Signal(syscall.SIGTERM) // signal shutdown
		_, err := w.Write([]byte("Shutting down..."))
		if err != nil {
			slog.Error("Error", err)
		}
	})

	slog.Info("🌏 serving prometheus metrics..", "address", fmt.Sprintf("http://%s:%d", ip, port))
	// Serve Prometheus metrics in a separate goroutine.
	//shutdownMetrics = metrics.ServeMetrics(promPort, metrics.END_TO_END_LATENCY, metrics.ONION_SIZE, metrics.LATENCY_BETWEEN_HOPS, metrics.PROCESSING_TIME, metrics.ONIONS_RECEIVED, metrics.ONIONS_SENT)

	// Start the HTTP server
	go func() {
		// Construct the address for the HTTP server based on the client's port.
		address := fmt.Sprintf(":%d", port)
		// Attempt to start the HTTP server.
		if err2 := http.ListenAndServe(address, nil); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", err2)
		}
	}()

	slog.Info("🌏 start newClient...", "address", fmt.Sprintf("http://%s:%d", ip, port))

	// Wait for either an OS signal to quit or the global context to be canceled
	select {
	case v := <-quit: // OS signal is received
		slog.Info("", "signal.Notify", v)
		config.GlobalCancel()
		cleanup()
	case done := <-config.GlobalCtx.Done(): // global context is canceled
		slog.Info("", "ctx.Done", done)
		cleanup()
	}

}

func cleanup() {
	//shutdownMetrics()
	newClient.ShutdownMetrics()
	stopNTC()
}
