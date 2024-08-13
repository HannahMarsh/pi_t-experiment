package bulletin_board

import (
	"github.com/HannahMarsh/pi_t-experiment/internal/api/structs"
	"github.com/HannahMarsh/pi_t-experiment/pkg/utils"
	"github.com/google/uuid"
)

type Checkpoint struct {
	Sender structs.PublicNodeApi
	Node   structs.PublicNodeApi
	Nonce  string
	Layer  int
}

func (c Checkpoint) GetAPI() structs.Checkpoint {
	return structs.Checkpoint{
		Receiver: c.Node,
		Nonce:    c.Nonce,
		Layer:    c.Layer,
	}
}

func generateUUIDs(n int) []string {
	uuids := make([]string, n)
	for i := 0; i < n; i++ {
		uuids[i] = uuid.New().String()
	}
	return uuids
}

func GetCheckpoints(nodes []structs.PublicNodeApi, clients []structs.PublicNodeApi, layers int, numMessages int) map[int][]structs.Checkpoint {
	checkpoints := make(map[int][]Checkpoint)
	for i := 1; i <= layers; i++ {
		checkpoints[i] = make([]Checkpoint, 0)
	}

	for _, client := range clients {
		for i := 1; i <= numMessages; i++ {
			checkpointLayers := utils.GenerateRandomBoolArray(layers/2, layers-(layers/2))
			uuids := generateUUIDs(layers)
			nonces := make([]string, layers)
			for j, hasCheckpoint := range checkpointLayers {
				if hasCheckpoint {
					nonces[j] = uuids[j]
				} else {
					nonces[j] = "" // Use empty string for no checkpoint
				}
			}
			for j := 0; j < layers; j++ {
				if checkpointLayers[j] {
					checkpoints[j] = append(checkpoints[j], Checkpoint{
						Sender: client,
						Node:   nodes[i],
						Nonce:  nonces[j],
						Layer:  j,
					})
				}
			}
		}
	}

	checkPointsMap := make(map[int][]structs.Checkpoint)
	for _, client := range clients {
		checkPointsMap[client.ID] = make([]structs.Checkpoint, 0)
	}

	for _, ch := range checkpoints {
		if ch != nil && len(ch) > 0 {
			checkPointsMap[ch[0].Sender.ID] = append(checkPointsMap[ch[0].Sender.ID], ch[0].GetAPI())
		}
	}

	return checkPointsMap
}
