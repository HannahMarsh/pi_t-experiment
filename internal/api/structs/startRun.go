package structs

type StartRunApi struct {
	ParticipatingClients []PublicNodeApi
	Mixers               []PublicNodeApi
	Gatekeepers          []PublicNodeApi
	NumMessagesPerClient int
}
