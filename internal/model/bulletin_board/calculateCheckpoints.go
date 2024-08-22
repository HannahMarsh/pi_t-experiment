package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"github.com/google/uuid"
)

// generateUUIDs creates a slice of UUID strings with a length of n.
func generateUUIDs(n int) []string {
	uuids := make([]string, n)
	for i := 0; i < n; i++ {
		uuids[i] = uuid.New().String()
	}
	return uuids
}

// GetCheckpoints generates checkpoint onions for each client based on the list of nodes and clients.
func GetCheckpoints(nodes, clients []structs.PublicNodeApi) map[int][]structs.CheckpointOnion {
	// Initialize a map to store the checkpoint onions for each client, keyed by client ID.
	checkpoints := make(map[int][]structs.CheckpointOnion)

	numClients := len(clients)
	numNodes := len(nodes)

	// Calculate the expected number of checkpoint onions each client should send based on the server load.
	expectedToSend := int((float64(numNodes)*float64(config.GlobalConfig.ServerLoad))/float64(numClients)) - 1

	for _, client := range clients {
		checkpoints[client.ID] = make([]structs.CheckpointOnion, 0)

		for i := 0; i < expectedToSend; i++ {
			path := make([]structs.Checkpoint, 0)

			// Generate the relay path for the checkpoint onion, which includes L1 mixers and L2 gatekeepers.
			for j := 0; j < config.GlobalConfig.L1+config.GlobalConfig.L2; j++ {
				path = append(path, structs.Checkpoint{
					Receiver: utils.RandomElement(nodes), // Randomly select a node as the receiver for this layer.
					Nonce:    uuid.New().String(),        // Generate a new UUID to use as the nonce for this layer.
					Layer:    j + 1,                      // Set the layer number, starting from 1.
				})
			}

			// Append the constructed checkpoint onion to the client's list of checkpoints.
			checkpoints[client.ID] = append(checkpoints[client.ID], structs.CheckpointOnion{
				Path: path, // Set the path for this checkpoint onion.
			})
		}
	}

	return checkpoints
}
