package structs

type OnionApi struct {
	To        string
	From      string
	Onion     string
	SharedKey string // TODO remove this, figure out how bulletin board will distribute the shared keys for the processing node/clients
}
