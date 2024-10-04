package structs

import "github.com/HannahMarsh/pi_t-experiment/config"

type ClientStartRunApi struct {
	Relays           []PublicNodeApi   `json:"r"`
	Clients          []PublicNodeApi   `json:"c"`
	CheckpointOnions []CheckpointOnion `json:"co"`
	Config           config.Config     `json:"cfg"`
}

type RelayStartRunApi struct {
	Checkpoints []Checkpoint  `json:"cp"`
	Config      config.Config `json:"cfg"`
}

type CheckpointOnion struct {
	Path []Checkpoint `json:"p"`
}

type Checkpoint struct {
	Receiver PublicNodeApi `json:"r"`
	Nonce    string        `json:"n"`
	Layer    int           `json:"l"`
}
