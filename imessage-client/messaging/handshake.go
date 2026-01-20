package messaging

import (
	"context"

	"imessage-client/config"
)

// Handshaker abstracts the IDS/NAC handshake. It should return a populated
// handshakeState or an error.
type Handshaker interface {
	Handshake(ctx context.Context, reg *config.RegistrationData) (*handshakeState, error)
}

// DefaultHandshaker is a stub that reports unimplemented.
type DefaultHandshaker struct{}

func (DefaultHandshaker) Handshake(_ context.Context, reg *config.RegistrationData) (*handshakeState, error) {
	if reg == nil {
		return nil, ErrInvalidRegistrationData
	}
	if len(reg.ValidationData) == 0 {
		return nil, ErrInvalidRegistrationData
	}
	return &handshakeState{ValidationData: reg.ValidationData, DeviceInfo: reg.DeviceInfo}, nil
}
