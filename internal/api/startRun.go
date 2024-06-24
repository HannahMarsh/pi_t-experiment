package api

type StartRunApi struct {
	ParticipatingClients []PublicNodeApi
	ActiveNodes          []PublicNodeApi
	NumMessagesPerClient int
}
