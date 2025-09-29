package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	pl "github.com/HannahMarsh/PrettyLogger"

	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/crypto/keys"
	"github.com/HannahMarsh/pi_t-experiment/internal/pi_t/onion_model"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
)

// Run with: go run ./internal/pi_t/measurement_harness
func main() {
	pl.SetUpLogrusAndSlog("debug")

	// CSV output file
	f, err := os.Create("internal/pi_t/measurement_harness/output/onion_sizes.csv")
	if err != nil {
		slog.Error("failed to create CSV", err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			slog.Error("failed to close file", err)
		}
	}(f)

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err = writer.Write([]string{"PathLength", "MessageSize", "TotalBytes", "HeaderBytes", "SepalBytes", "ContentBytes"}); err != nil {
		slog.Error("Failed to write to writer", err)
		os.Exit(-1)
	}

	pathLengths := []int{3, 5, 7}
	messageSizes := []int{64, 256, 1024}

	for _, l := range pathLengths {
		for _, m := range messageSizes {
			measureOne(l, m, writer)
		}
	}
}

func measureOne(pathLength int, msgSize int, writer *csv.Writer) {
	// ---- Setup nodes, keys, routing path ----
	nodes := make([]struct {
		priv string
		pub  string
		addr string
	}, pathLength+1)

	for i := 0; i < pathLength+1; i++ {
		priv, pub, err := keys.KeyGen()
		if err != nil {
			slog.Error("KeyGen failed", err)
			os.Exit(-1)
		}
		nodes[i].priv = priv
		nodes[i].pub = pub
		nodes[i].addr = fmt.Sprintf("relay%d", i)
	}

	payload := make([]byte, msgSize)
	for i := 0; i < msgSize; i++ {
		payload[i] = byte('A' + (i % 26))
	}

	msg, _ := json.Marshal(structs.Message{
		Msg:  string(payload),
		To:   nodes[pathLength].addr,
		From: nodes[0].addr,
	})

	publicKeys := utils.Map(nodes[1:], func(n struct {
		priv string
		pub  string
		addr string
	}) string {
		return n.pub
	})
	routingPath := utils.Map(nodes[1:], func(n struct {
		priv string
		pub  string
		addr string
	}) string {
		return n.addr
	})

	metadata := make([]onion_model.Metadata, pathLength+1)

	// ---- Form the onion ----
	onionLayers, err := pi_t.FORMONION(string(msg), routingPath[:pathLength-1], []string{}, routingPath[len(routingPath)-1], publicKeys, metadata, 2)
	if err != nil {
		slog.Error("FORMONION failed", err)
		os.Exit(-1)
	}

	o := onionLayers[0][0] // just take the first onion

	// ---- Measure sizes ----
	headerBytes, _ := json.Marshal(o.Header)
	sepalBytes, _ := json.Marshal(o.Sepal)
	contentBytes := []byte(o.Content)

	totalBytes, _ := json.Marshal(o)

	if err = writer.Write([]string{
		fmt.Sprint(pathLength),
		fmt.Sprint(msgSize),
		fmt.Sprint(len(totalBytes)),
		fmt.Sprint(len(headerBytes)),
		fmt.Sprint(len(sepalBytes)),
		fmt.Sprint(len(contentBytes)),
	}); err != nil {
		slog.Error("Failed to write to writer", err)
		os.Exit(-1)
	}

	fmt.Printf("l=%d, msg=%dB -> total=%dB (header=%d, sepal=%d, content=%d)\n",
		pathLength, msgSize,
		len(totalBytes), len(headerBytes), len(sepalBytes), len(contentBytes))
}
