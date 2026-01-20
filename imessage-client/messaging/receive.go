package messaging

import (
	"context"
	"time"
)

// FetchMessages drains accumulated messages from APNS.
func (s *Session) FetchMessages(ctx context.Context) ([]Message, error) {
	if err := s.ensureHandshake(); err != nil {
		return nil, err
	}

	// Drain all messages from the channel
	var messages []Message
	for {
		select {
		case msg := <-s.messageChan:
			if msg != nil {
				messages = append(messages, *msg)
			}
		default:
			// No more messages
			return messages, nil
		}
	}
}

// filterUnread compares fetched messages against store to emit only new ones.
func (s *Session) filterUnread(messages []Message) []Message {
	var unread []Message
	for _, msg := range messages {
		lastSeen := s.store.LastSeen(msg.Chat)
		if msg.Timestamp.After(lastSeen) {
			unread = append(unread, msg)
		}
	}
	return unread
}

// updateStore marks messages as seen per chat.
func (s *Session) updateStore(messages []Message) error {
	chatLatest := make(map[string]time.Time)
	for _, msg := range messages {
		if existing, ok := chatLatest[msg.Chat]; !ok || msg.Timestamp.After(existing) {
			chatLatest[msg.Chat] = msg.Timestamp
		}
	}
	for chat, ts := range chatLatest {
		if err := s.store.SetLastSeen(chat, ts); err != nil {
			return err
		}
	}
	return nil
}
