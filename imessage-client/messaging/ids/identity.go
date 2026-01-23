package ids

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
)

// UserIdentity represents a user's public identity for iMessage.
type UserIdentity struct {
	SigningKey    *ecdsa.PublicKey
	EncryptionKey *rsa.PublicKey
}

// asnIdentity is the ASN.1 structure for encoding identity.
type asnIdentity struct {
	SigningKey    []byte `asn1:"tag:1"`
	EncryptionKey []byte `asn1:"tag:2"`
}

// marshalSigningKey encodes the ECDSA public key with Apple's format.
func (i *UserIdentity) marshalSigningKey() []byte {
	return append([]byte{0x00, 0x41}, elliptic.Marshal(elliptic.P256(), i.SigningKey.X, i.SigningKey.Y)...)
}

// marshalEncryptionKey encodes the RSA public key with Apple's format.
func (i *UserIdentity) marshalEncryptionKey() []byte {
	return append([]byte{0x00, 0xAC}, x509.MarshalPKCS1PublicKey(i.EncryptionKey)...)
}

// ToBytes serializes the identity to bytes for registration.
func (i *UserIdentity) ToBytes() []byte {
	out, err := asn1.Marshal(asnIdentity{
		SigningKey:    i.marshalSigningKey(),
		EncryptionKey: i.marshalEncryptionKey(),
	})
	if err != nil {
		panic(err)
	}
	return out
}
