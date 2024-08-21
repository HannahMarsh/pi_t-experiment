package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/config"
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"github.com/google/uuid"
)

func generateUUIDs(n int) []string {
	uuids := make([]string, n)
	for i := 0; i < n; i++ {
		uuids[i] = uuid.New().String()
	}
	return uuids
}

func GetCheckpoints(nodes, clients []structs.PublicNodeApi) map[int][]structs.CheckpointOnion {
	checkpoints := make(map[int][]structs.CheckpointOnion)

	numClients := len(clients)
	numNodes := len(nodes)

	expectedToSend := int((float64(numNodes)*float64(config.GlobalConfig.ServerLoad))/float64(numClients)) - 1

	for _, client := range clients {
		checkpoints[client.ID] = make([]structs.CheckpointOnion, 0)
		for i := 0; i < expectedToSend; i++ {
			path := make([]structs.Checkpoint, 0)
			for j := 0; j < config.GlobalConfig.L1+config.GlobalConfig.L2; j++ {
				path = append(path, structs.Checkpoint{
					Receiver: utils.RandomElement(nodes),
					Nonce:    uuid.New().String(),
					Layer:    j + 1,
				})
			}
			checkpoints[client.ID] = append(checkpoints[client.ID], structs.CheckpointOnion{
				Path: path,
			})
		}
	}

	return checkpoints
}
