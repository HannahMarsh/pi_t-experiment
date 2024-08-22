package main

import (
	"flag"
	"fmt"
	pl "github.com/HannahMarsh/PrettyLogger"
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"go.uber.org/automaxprocs/maxprocs"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
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

	pl.SetUpLogrusAndSlog(*logLevel)

	// Automatically adjust the GOMAXPROCS setting based on the number of available CPU cores.
	if _, err := maxprocs.Set(); err != nil {
		slog.Error("failed set max procs", err)
		os.Exit(1)
	}

	if err := config.InitGlobal(); err != nil {
		slog.Error("failed to init config", err)
		os.Exit(1)
	}

	nodePromAddresses := utils.Map(config.GlobalConfig.Relays, func(n config.Relay) string {
		return fmt.Sprintf("http://%s:%d/metrics", n.Host, n.PrometheusPort)
	})

	clientPromAddresses := utils.Map(config.GlobalConfig.Clients, func(c config.Client) string {
		return fmt.Sprintf("http://%s:%d/metrics", c.Host, c.PrometheusPort)
	})

	slog.Info("⚡ init visualizer", "nodePromAddresses", nodePromAddresses, "clientPromAddresses", clientPromAddresses)

	scrapeInterval := time.Duration(config.GlobalConfig.ScrapeInterval) * time.Millisecond

	// Start the metric collector
	for {
		nextScrape := time.Now().Add(scrapeInterval)
		var wg sync.WaitGroup
		wg.Add(len(nodePromAddresses) + len(clientPromAddresses))

		for _, address := range nodePromAddresses {
			go func(address string) {
				defer wg.Done()
				scrapeMetrics(address)
			}(address)
		}

		for _, address := range clientPromAddresses {
			go func(address string) {
				defer wg.Done()
				scrapeMetrics(address)
			}(address)
		}

		wg.Wait()
		if time.Until(nextScrape) > 0 {
			time.Sleep(time.Until(nextScrape))
		}
	}

}

func scrapeMetrics(address string) {
	resp, err := http.Get(address)
	if err != nil {
		slog.Error("failed to scrape visualizer", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		pl.LogNewError("%s: unexpected status code %d", address, resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body", err)
		return
	}

	slog.Debug("scraped visualizer", "address", address, "response", string(body))
}
