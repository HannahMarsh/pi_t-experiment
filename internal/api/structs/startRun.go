package structs

type ClientStartRunApi struct {
	Relays           []PublicNodeApi
	Clients          []PublicNodeApi
	CheckpointOnions []CheckpointOnion
}

type RelayStartRunApi struct {
	Checkpoints []Checkpoint
}

type CheckpointOnion struct {
	Path []Checkpoint
}

type Checkpoint struct {
	Receiver PublicNodeApi
	Nonce    string
	Layer    int
}
