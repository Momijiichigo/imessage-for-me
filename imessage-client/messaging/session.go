package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"imessage-client/config"
	"imessage-client/messaging/apns"
	"imessage-client/messaging/ids"
)

// Session represents an authenticated connection to Apple's iMessage services.
type Session struct {
	registration *config.RegistrationData
	store        Store
	state        *handshakeState
	handshaker   Handshaker

	// APNS message accumulation
	messageChan    chan *Message
	readLoopCtx    context.Context
	readLoopCancel context.CancelFunc
}

// Connect validates registration data and establishes a session (stubbed for now).
func Connect(_ context.Context, reg *config.RegistrationData, store Store) (*Session, error) {
	if reg == nil {
		return nil, errors.New("registration data is nil")
	}
	if len(reg.ValidationData) == 0 {
		return nil, ErrInvalidRegistrationData
	}
	if reg.IsExpired() {
		return nil, ErrRegistrationExpired
	}
	if store == nil {
		store = NewMemoryStore()
	}
	// Use RealHandshaker instead of stub
	return &Session{registration: reg, store: store, handshaker: RealHandshaker{}}, nil
}

// FetchUnread will retrieve unread messages once the transport is implemented.
func (s *Session) FetchUnread(ctx context.Context) ([]MessageSummary, error) {
	if s == nil {
		return nil, errors.New("session is nil")
	}
	if err := s.ensureHandshake(); err != nil {
		return nil, err
	}

	// Fetch all available messages
	messages, err := s.FetchMessages(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to only unread
	unread := s.filterUnread(messages)

	// Update store with what we've seen
	if err := s.updateStore(unread); err != nil {
		return nil, err
	}

	// Convert to summaries
	var summaries []MessageSummary
	for _, msg := range unread {
		summaries = append(summaries, msg.ToSummary())
	}

	return summaries, nil
}

// Close cleans up session resources.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}

	// Stop APNS read loop
	if s.readLoopCancel != nil {
		s.readLoopCancel()
	}

	// Close APNS connection
	if s.state != nil && s.state.APNSConn != nil {
		return s.state.APNSConn.Close()
	}

	return nil
}

// startAPNS connects to APNS and starts the message read loop.
func (s *Session) startAPNS(ctx context.Context) error {
	conn := s.state.APNSConn

	// TODO: Get actual push token from certificate generation
	// For now, this will fail with ErrNoToken - that's expected until
	// we implement the full NAC authentication flow

	// Connect to APNS
	if err := conn.Connect(ctx); err != nil {
		return err
	}

	// Set message handler to accumulate messages
	conn.SetMessageHandler(s.handleAPNSMessage)

	// Subscribe to iMessage topics
	if err := conn.Filter(apns.TopicMadrid); err != nil {
		return err
	}

	// Set connection to active state
	if err := conn.SetState(1); err != nil {
		return err
	}

	// Start read loop in background
	s.readLoopCtx, s.readLoopCancel = context.WithCancel(context.Background())
	go func() {
		if err := conn.ReadLoop(s.readLoopCtx); err != nil {
			fmt.Printf("APNS read loop ended: %v\n", err)
		}
	}()

	return nil
}

// handleAPNSMessage processes incoming APNS messages and accumulates them.
func (s *Session) handleAPNSMessage(ctx context.Context, payload *apns.SendMessagePayload) error {
	// Try to decrypt the message
	if s.state == nil || s.state.IDSConfig == nil || s.state.IDSConfig.IDSEncryptionKey == nil {
		// No encryption key available, create stub
		msg := &Message{
			ID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
			Chat:      "unknown-chat",
			Sender:    "unknown-sender",
			Text:      fmt.Sprintf("[Encrypted] %d bytes from %s", len(payload.Payload), payload.Topic),
			Timestamp: time.Now(),
		}

		select {
		case s.messageChan <- msg:
			return nil
		default:
			return fmt.Errorf("message channel full")
		}
	}

	// Attempt decryption
	imsg, err := DecryptMessage(s.state.IDSConfig.IDSEncryptionKey, payload.Payload)
	if err != nil {
		// Decryption failed, still accumulate as encrypted message
		msg := &Message{
			ID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
			Chat:      "unknown-chat",
			Sender:    "unknown-sender",
			Text:      fmt.Sprintf("[Decrypt failed: %s] %d bytes", err.Error(), len(payload.Payload)),
			Timestamp: time.Now(),
		}

		select {
		case s.messageChan <- msg:
			return fmt.Errorf("decryption failed: %w", err)
		default:
			return fmt.Errorf("message channel full")
		}
	}

	// Successfully decrypted!
	chat := "direct"
	if imsg.GroupID != "" {
		chat = imsg.GroupID
	} else if len(imsg.Participants) > 0 {
		chat = imsg.Participants[0]
	}

	sender := "unknown"
	if len(imsg.Participants) > 0 {
		sender = imsg.Participants[0]
	}

	msgID := imsg.MessageUUID
	if msgID == "" {
		msgID = fmt.Sprintf("msg-%d", time.Now().Unix())
	}

	msg := &Message{
		ID:        msgID,
		Chat:      chat,
		Sender:    sender,
		Text:      imsg.Text,
		Timestamp: time.Now(),
	}

	select {
	case s.messageChan <- msg:
		return nil
	default:
		return fmt.Errorf("message channel full")
	}
}

// handshakeState will hold derived keys/session tokens once the IDS/NAC flow is ported.
type handshakeState struct {
	ValidationData []byte
	DeviceInfo     config.DeviceInfo
	IDSConfig      *ids.Config
	APNSConn       *apns.Connection
}

func (s *Session) ensureHandshake() error {
	if s.state != nil {
		return nil
	}
	if s.handshaker == nil {
		return ErrHandshakeNotImplemented
	}
	state, err := s.handshaker.Handshake(context.Background(), s.registration)
	if err != nil {
		return err
	}
	s.state = state
	return nil
}
