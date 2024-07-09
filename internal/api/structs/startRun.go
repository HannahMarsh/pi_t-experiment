package structs

type ClientStartRunApi struct {
	Mixers               []PublicNodeApi
	Gatekeepers          []PublicNodeApi
	Clients              []PublicNodeApi
	NumMessagesPerClient int
	Checkpoints          [][]int
}

type NodeStartRunApi struct {
	Mixers      []PublicNodeApi
	Gatekeepers []PublicNodeApi
	Clients     []PublicNodeApi
	Checkpoints [][]int
}
