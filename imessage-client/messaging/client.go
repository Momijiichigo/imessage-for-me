package messaging

import (
	"context"
	"time"

	"imessage-client/config"
)

type MessageSummary struct {
	Sender    string
	Preview   string
	Timestamp time.Time
}

type Client struct {
	registration *config.RegistrationData
	store        Store
}

func NewClient(reg *config.RegistrationData) *Client {
	return &Client{registration: reg, store: NewMemoryStore()}
}

// NewClientWithStore allows the caller to provide a persistent Store implementation.
func NewClientWithStore(reg *config.RegistrationData, store Store) *Client {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Client{registration: reg, store: store}
}

func (c *Client) PollUnread(ctx context.Context) ([]MessageSummary, error) {
	session, err := Connect(ctx, c.registration, c.store)
	if err != nil {
		return nil, err
	}
	return session.FetchUnread(ctx)
}
