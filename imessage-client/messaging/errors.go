package messaging

import "errors"

var (
	ErrNotImplemented          = errors.New("not implemented")
	ErrRegistrationExpired     = errors.New("registration expired")
	ErrInvalidRegistrationData = errors.New("registration data missing required fields")
	ErrHandshakeNotImplemented = errors.New("handshake not implemented")
)
