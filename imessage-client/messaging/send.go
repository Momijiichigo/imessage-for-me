package messaging

import "context"

// Send sends a message to the given chat/recipient. Currently a stub.
func (c *Client) Send(ctx context.Context, chat string, text string) error {
	session, err := Connect(ctx, c.registration, c.store)
	if err != nil {
		return err
	}
	if err := session.ensureHandshake(); err != nil {
		return err
	}
	// TODO: implement actual send using APNS/IDS
	return ErrNotImplemented
}
