package messaging

import (
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"

	"howett.net/plist"
)

var normalIV = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// ParsedBody represents a parsed encrypted message body.
type ParsedBody struct {
	Tag       byte
	Body      []byte
	Signature []byte
}

// ParseBody parses the encrypted payload structure.
// Format: [tag:1byte][bodyLen:2bytes][body:bodyLen][sigLen:1byte][signature:sigLen]
func ParseBody(payload []byte) (ParsedBody, error) {
	if len(payload) < 4 {
		return ParsedBody{}, fmt.Errorf("too short payload (missing header, expected >4, got %d)", len(payload))
	}

	tag := payload[0]
	bodyLength := binary.BigEndian.Uint16(payload[1:3])

	expectedLength := int(3 + bodyLength + 1)
	if len(payload) < expectedLength {
		return ParsedBody{}, fmt.Errorf("too short payload (missing body, expected >%d, got %d)", expectedLength, len(payload))
	}

	body := payload[3 : 3+bodyLength]
	signatureLength := payload[3+bodyLength]

	expectedLength = int(3 + bodyLength + 1 + uint16(signatureLength))
	if len(payload) < expectedLength {
		return ParsedBody{}, fmt.Errorf("too short payload (missing signature, expected %d, got %d)", expectedLength, len(payload))
	}

	signature := payload[3+bodyLength+1:]

	return ParsedBody{
		Tag:       tag,
		Body:      body,
		Signature: signature,
	}, nil
}

// DecryptPairPayload decrypts a "pair" encrypted message using RSA+AES.
func DecryptPairPayload(privateKey *rsa.PrivateKey, body ParsedBody) ([]byte, error) {
	if len(body.Body) < 160 {
		return nil, fmt.Errorf("too short payload (missing encryption key, expected >160, got %d)", len(body.Body))
	}

	// First 160 bytes are RSA-encrypted AES key + first 100 bytes of ciphertext
	encryptedEncryptionKey := body.Body[:160]
	// Rest is the remaining ciphertext
	actualEncryptedMessage := body.Body[160:]

	// Decrypt the RSA part to get AES key + first part of message
	decryptedEncryptionKey, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, privateKey, encryptedEncryptionKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt encryption key: %w", err)
	}

	// AES key is first 16 bytes, then comes the first part of encrypted payload
	block, err := aes.NewCipher(decryptedEncryptionKey[:16])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Combine the decrypted first part with the rest
	decryptionInput := append(decryptedEncryptionKey[16:], actualEncryptedMessage...)

	// Decrypt using AES-CTR
	cipher.NewCTR(block, normalIV).XORKeyStream(decryptionInput, decryptionInput)

	return decryptionInput, nil
}

// MaybeGUnzip attempts to decompress gzipped data, or returns original if not gzipped.
func MaybeGUnzip(data []byte) ([]byte, error) {
	// Check for gzip magic number
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		// Not gzipped, return as-is
		return data, nil
	}

	// Decompress
	r, err := gzip.NewReader(io.NopCloser(&readerWrapper{data: data}))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	return decompressed, nil
}

// readerWrapper wraps a byte slice as io.Reader
type readerWrapper struct {
	data []byte
	pos  int
}

func (r *readerWrapper) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// IMessagePayload represents the decrypted iMessage payload structure.
type IMessagePayload struct {
	// Basic message fields
	Text         string   `plist:"t,omitempty"` // Message text
	Subject      string   `plist:"s,omitempty"` // Message subject (rare)
	Participants []string `plist:"p,omitempty"` // Chat participants

	// Metadata
	GroupID     string `plist:"gid,omitempty"` // Group chat ID
	Version     int    `plist:"v,omitempty"`   // Protocol version
	MessageUUID string `plist:"r,omitempty"`   // Message UUID (reply-to)

	// Additional fields can be added as needed
	// See imessage/imessage/direct/decrypt.go for full structure
}

// DecryptMessage decrypts and parses an iMessage from APNS payload.
func DecryptMessage(privateKey *rsa.PrivateKey, payload []byte) (*IMessagePayload, error) {
	// Step 1: Parse the encrypted body structure
	parsed, err := ParseBody(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body: %w", err)
	}

	// Step 2: Decrypt using RSA+AES (pair encryption)
	decrypted, err := DecryptPairPayload(privateKey, parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Step 3: Decompress if gzipped
	decompressed, err := MaybeGUnzip(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	// Step 4: Parse plist
	var msg IMessagePayload
	if _, err := plist.Unmarshal(decompressed, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plist: %w", err)
	}

	return &msg, nil
}
