package apns

import (
	"encoding/binary"
	"fmt"
	"io"
)

// CommandID identifies different APNS commands.
type CommandID uint8

const (
	CommandConnect         CommandID = 7
	CommandConnectAck      CommandID = 8
	CommandFilterTopics    CommandID = 9
	CommandSendMessage     CommandID = 10
	CommandSendMessageAck  CommandID = 11
	CommandKeepAlive       CommandID = 12
	CommandKeepAliveAck    CommandID = 13
	CommandFilterTopicsAck CommandID = 14
	CommandSetState        CommandID = 20
)

// FieldID identifies fields within APNS commands.
type FieldID uint8

// Field represents a TLV (type-length-value) field in APNS protocol.
type Field struct {
	ID    FieldID
	Value []byte
}

// Payload represents an APNS command with its fields.
type Payload struct {
	ID     CommandID
	Fields []Field
}

// FindField returns the value of a field by ID, or nil if not found.
func (p *Payload) FindField(fieldID uint8) []byte {
	for _, field := range p.Fields {
		if field.ID == FieldID(fieldID) {
			return field.Value
		}
	}
	return nil
}

// ToBytes serializes the payload to binary format.
// Format: [CommandID:1][Length:4][Fields...]
// Each field: [FieldID:1][FieldLength:2][FieldValue...]
func (p *Payload) ToBytes() []byte {
	// Calculate total length
	length := 5 // 1 byte cmd + 4 bytes length
	for _, field := range p.Fields {
		length += 3 + len(field.Value) // 1 byte ID + 2 bytes length + value
	}

	payload := make([]byte, length)
	payload[0] = byte(p.ID)
	binary.BigEndian.PutUint32(payload[1:5], uint32(length-5))

	ptr := 5
	for _, field := range p.Fields {
		payload[ptr] = byte(field.ID)
		binary.BigEndian.PutUint16(payload[ptr+1:ptr+3], uint16(len(field.Value)))
		copy(payload[ptr+3:ptr+3+len(field.Value)], field.Value)
		ptr += 3 + len(field.Value)
	}

	return payload
}

// UnmarshalBinaryStream reads a payload from a stream.
func (p *Payload) UnmarshalBinaryStream(reader io.Reader) error {
	// Read command ID
	readBuf := make([]byte, 4)
	if _, err := io.ReadFull(reader, readBuf[:1]); err != nil {
		return err
	}
	p.ID = CommandID(readBuf[0])
	if p.ID == 0 {
		return nil
	}

	// Read payload length
	if _, err := io.ReadFull(reader, readBuf); err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(readBuf)

	// Read full payload
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return err
	}

	return p.unmarshalFieldsFromBytes(data)
}

// UnmarshalBinary deserializes a payload from binary format.
func (p *Payload) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty payload")
	}
	p.ID = CommandID(data[0])
	if p.ID == 0 {
		return nil
	}
	if len(data) < 5 {
		return fmt.Errorf("invalid payload length")
	}
	length := binary.BigEndian.Uint32(data[1:5])
	return p.unmarshalFieldsFromBytes(data[5 : 5+length])
}

// unmarshalFieldsFromBytes parses TLV fields from bytes.
func (p *Payload) unmarshalFieldsFromBytes(data []byte) error {
	i := 0
	for i+3 <= len(data) {
		var field Field
		field.ID = FieldID(data[i])
		fieldLength := int(binary.BigEndian.Uint16(data[i+1 : i+3]))

		if i+3+fieldLength > len(data) {
			return fmt.Errorf("invalid field length")
		}

		field.Value = data[i+3 : i+3+fieldLength]
		i += 3 + fieldLength
		p.Fields = append(p.Fields, field)
	}
	return nil
}

// ConnectionFlag represents APNS connection flags.
type ConnectionFlag uint32

const (
	BaseConnectionFlags ConnectionFlag = 0b1000001
	RootConnection      ConnectionFlag = 0b100
)

// ToBytes converts flags to big-endian bytes.
func (c ConnectionFlag) ToBytes() []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(c))
	return out
}
