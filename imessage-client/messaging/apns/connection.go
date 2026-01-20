package apns

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	mathrand "math/rand"
	"net"
	"time"
)

var (
	ErrNotConnected = errors.New("not connected to APNS")
	ErrNoToken      = errors.New("no push token available")
)

// MessageHandler processes incoming messages from APNS.
type MessageHandler func(ctx context.Context, payload *SendMessagePayload) error

// SendMessagePayload represents an incoming APNS message.
type SendMessagePayload struct {
	Topic   string
	Payload []byte
}

// Connection represents an APNS courier connection for iMessage.
type Connection struct {
	privateKey *rsa.PrivateKey
	deviceCert *x509.Certificate
	token      []byte

	conn           net.Conn
	messageHandler MessageHandler

	maxMessageSize      int
	maxLargeMessageSize int
}

// NewConnection creates a new APNS connection.
func NewConnection(privateKey *rsa.PrivateKey, deviceCert *x509.Certificate, token []byte) *Connection {
	return &Connection{
		privateKey:          privateKey,
		deviceCert:          deviceCert,
		token:               token,
		maxMessageSize:      4 * 1024,
		maxLargeMessageSize: 15 * 1024,
	}
}

// SetMessageHandler sets the handler for incoming messages.
func (c *Connection) SetMessageHandler(handler MessageHandler) {
	c.messageHandler = handler
}

// Connect establishes a TLS connection to Apple's push courier.
func (c *Connection) Connect(ctx context.Context) error {
	if c.privateKey == nil || c.deviceCert == nil {
		return fmt.Errorf("missing push certificate or key")
	}
	if len(c.token) == 0 {
		return ErrNoToken
	}

	// Get courier hostname (randomly select from 1-50)
	hostNum := mathrand.Intn(CourierHostCount) + 1
	host := fmt.Sprintf("%d-%s", hostNum, CourierHostname)
	addr := fmt.Sprintf("%s:%d", host, CourierPort)

	// Setup TLS config
	tlsConfig := &tls.Config{
		ServerName: CourierHostname,
		NextProtos: []string{"apns-security-v3"},
		MinVersion: tls.VersionTLS12,
	}

	// Connect with TLS
	dialer := &tls.Dialer{Config: tlsConfig}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to dial APNS: %w", err)
	}
	c.conn = conn

	// Send connect command with signed nonce
	nonce := make([]byte, 20)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce[0] = 0 // First byte must be 0

	sum := sha1.Sum(nonce)
	signature, err := rsa.SignPKCS1v15(nil, c.privateKey, crypto.SHA1, sum[:])
	if err != nil {
		return fmt.Errorf("failed to sign nonce: %w", err)
	}

	// Build connect command
	connectCmd := &ConnectCommand{
		DeviceToken: c.token,
		State:       []byte{1},
		Flags:       BaseConnectionFlags | RootConnection,
		Cert:        c.deviceCert.Raw,
		Nonce:       nonce,
		Signature:   append([]byte{0x1, 0x1}, signature...),
	}

	// Send connect
	if err := c.write(connectCmd.ToPayload().ToBytes()); err != nil {
		return fmt.Errorf("failed to send connect command: %w", err)
	}

	// Wait for connect ack
	payload, err := c.readPayload()
	if err != nil {
		return fmt.Errorf("failed to read connect ack: %w", err)
	}

	if payload.ID != CommandConnectAck {
		return fmt.Errorf("unexpected response to connect: command %d", payload.ID)
	}

	var ack ConnectAckCommand
	ack.FromPayload(payload)

	// Check status (0 = success, 2 = error)
	if len(ack.Status) > 0 && ack.Status[0] != 0 {
		return fmt.Errorf("connection rejected by APNS, status: %x", ack.Status)
	}

	// Update token and limits
	if len(ack.Token) > 0 {
		c.token = ack.Token
	}
	if ack.MaxMessageSize > 0 {
		c.maxMessageSize = int(ack.MaxMessageSize)
	}
	if ack.LargeMessageSize > 0 {
		c.maxLargeMessageSize = int(ack.LargeMessageSize)
	}

	return nil
}

// Filter subscribes to specific APNS topics.
func (c *Connection) Filter(topics ...Topic) error {
	if c.conn == nil {
		return ErrNotConnected
	}

	sha1Topics := make([][]byte, len(topics))
	for i, topic := range topics {
		hashed := sha1.Sum([]byte(topic))
		sha1Topics[i] = hashed[:]
	}

	cmd := &FilterTopicsCommand{
		Token:  c.token,
		Topics: sha1Topics,
	}

	return c.write(cmd.ToPayload().ToBytes())
}

// SetState sets the connection state.
func (c *Connection) SetState(state uint8) error {
	if c.conn == nil {
		return ErrNotConnected
	}

	cmd := &SetStateCommand{
		State:    state,
		FieldTwo: 0x7fffffff,
	}

	return c.write(cmd.ToPayload().ToBytes())
}

// ReadLoop continuously reads and processes incoming messages.
func (c *Connection) ReadLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, err := c.readPayload()
		if err != nil {
			return fmt.Errorf("failed to read payload: %w", err)
		}

		switch payload.ID {
		case CommandSendMessage:
			if c.messageHandler != nil {
				var msg IncomingSendMessageCommand
				msg.FromPayload(payload)

				msgPayload := &SendMessagePayload{
					Topic:   string(msg.Topic),
					Payload: msg.Payload,
				}

				if err := c.messageHandler(ctx, msgPayload); err != nil {
					fmt.Printf("Error handling message: %v\n", err)
				}
			}

		case CommandKeepAlive:
			keepAlive := &KeepAliveCommand{}
			if err := c.write(keepAlive.ToPayload().ToBytes()); err != nil {
				return fmt.Errorf("failed to respond to keep-alive: %w", err)
			}

		case CommandConnectAck, CommandFilterTopicsAck, CommandSendMessageAck, CommandKeepAliveAck:
			// Responses we expect, ignore for now

		default:
			fmt.Printf("Received unknown command: %d\n", payload.ID)
		}
	}
}

// Close closes the APNS connection.
func (c *Connection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Connection) write(data []byte) error {
	if c.conn == nil {
		return ErrNotConnected
	}
	if err := c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	_, err := c.conn.Write(data)
	return err
}

func (c *Connection) readPayload() (*Payload, error) {
	if c.conn == nil {
		return nil, ErrNotConnected
	}

	payload := &Payload{}
	if err := payload.UnmarshalBinaryStream(c.conn); err != nil {
		return nil, err
	}
	return payload, nil
}
