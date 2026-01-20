package messaging

import (
	"context"
	"fmt"

	"imessage-client/config"
)

// NacIDSHandshaker is a placeholder for the real NAC/IDS handshake implementation.
// It sketches the shape we'll need once the Beeper code is pared down and ported.
type NacIDSHandshaker struct {
	// TODO: inject nacserv client, HTTP client, and minimal logger here
}

func (NacIDSHandshaker) Handshake(ctx context.Context, _ *config.RegistrationData) (*handshakeState, error) {
	return nil, fmt.Errorf("%w: NAC/IDS flow not wired", ErrHandshakeNotImplemented)
}
