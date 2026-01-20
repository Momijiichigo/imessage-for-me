package messaging

import "time"

// Message is a simplified iMessage payload representation for the CLI.
type Message struct {
	ID        string
	Chat      string
	Sender    string
	Text      string
	Timestamp time.Time
	Service   string
}

// ToSummary converts a full message to a MessageSummary for notifier output.
func (m Message) ToSummary() MessageSummary {
	return MessageSummary{
		Sender:    m.Sender,
		Preview:   m.Text,
		Timestamp: m.Timestamp,
	}
}
