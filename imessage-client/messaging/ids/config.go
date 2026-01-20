package ids

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"time"

	"github.com/google/uuid"
)

// Config holds the minimal IDS configuration needed for handshake.
// Stripped down from beeper/imessage/imessage/direct/ids/config.go
type Config struct {
	ProfileID      string
	AuthPrivateKey *rsa.PrivateKey

	AuthIDCertPairs map[string]*AuthIDCertPair
	IDRegisteredAt  time.Time

	PushKey   *rsa.PrivateKey
	PushCert  *x509.Certificate
	PushToken []byte

	IDSEncryptionKey *rsa.PrivateKey
	IDSSigningKey    *ecdsa.PrivateKey

	Handles       []ParsedURI
	DefaultHandle ParsedURI

	DeviceUUID      uuid.UUID
	LoggedInAt      time.Time
	HardwareVersion string
	SoftwareName    string
	SoftwareVersion string
	SoftwareBuildID string
}

type AuthIDCertPair struct {
	Added    time.Time
	AuthCert *x509.Certificate
	IDCert   *x509.Certificate

	RefreshNeeded bool
}

// ParsedURI is a simplified handle/identifier representation.
type ParsedURI struct {
	Scheme     string
	Identifier string
}

func (p ParsedURI) String() string {
	return p.Scheme + ":" + p.Identifier
}

func (p ParsedURI) IsEmpty() bool {
	return p.Scheme == "" && p.Identifier == ""
}

var EmptyURI = ParsedURI{}

const (
	SchemeTel   = "tel"
	SchemeEmail = "mailto"
)
