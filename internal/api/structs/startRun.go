package structs

type ClientStartRunApi struct {
	Mixers               []PublicNodeApi
	Gatekeepers          []PublicNodeApi
	Clients              []PublicNodeApi
	NumMessagesPerClient int
	Checkpoints          []Checkpoint
}

type NodeStartRunApi struct {
	Mixers      []PublicNodeApi
	Gatekeepers []PublicNodeApi
	Clients     []PublicNodeApi
	Checkpoints []Checkpoint
}

type Checkpoint struct {
	Receiver PublicNodeApi
	Nonce    string
	Layer    int
}
