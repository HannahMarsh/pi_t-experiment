package models

// Message represents a message to be sent through the onion routing network
type Message struct {
	ID      string
	Content string
}

// NewMessage creates a new message
func NewMessage(id, content string) *Message {
	return &Message{
		ID:      id,
		Content: content,
	}
}
