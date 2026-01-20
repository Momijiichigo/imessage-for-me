package apns

// ConnectCommand is sent to establish APNS connection.
type ConnectCommand struct {
	DeviceToken []byte
	State       []byte
	Flags       ConnectionFlag
	Cert        []byte
	Nonce       []byte
	Signature   []byte
}

// ToPayload converts ConnectCommand to binary payload.
func (c *ConnectCommand) ToPayload() *Payload {
	return &Payload{
		ID: CommandConnect,
		Fields: []Field{
			{ID: 1, Value: c.DeviceToken},
			{ID: 2, Value: c.State},
			{ID: 5, Value: c.Flags.ToBytes()},
			{ID: 12, Value: c.Cert},
			{ID: 13, Value: c.Nonce},
			{ID: 14, Value: c.Signature},
		},
	}
}

// ConnectAckCommand is received after successful connection.
type ConnectAckCommand struct {
	Status           []byte
	Token            []byte
	MaxMessageSize   uint16
	Unknown5         []byte
	Capabilities     []byte
	LargeMessageSize uint16
	ServerTimestamp  uint64
}

// FromPayload parses ConnectAckCommand from payload.
func (c *ConnectAckCommand) FromPayload(p *Payload) {
	c.Status = p.FindField(1)
	c.Token = p.FindField(3)
	if val := p.FindField(4); len(val) >= 2 {
		c.MaxMessageSize = uint16(val[0])<<8 | uint16(val[1])
	}
	c.Unknown5 = p.FindField(5)
	c.Capabilities = p.FindField(6)
	if val := p.FindField(8); len(val) >= 2 {
		c.LargeMessageSize = uint16(val[0])<<8 | uint16(val[1])
	}
	if val := p.FindField(10); len(val) >= 8 {
		c.ServerTimestamp = uint64(val[0])<<56 | uint64(val[1])<<48 |
			uint64(val[2])<<40 | uint64(val[3])<<32 |
			uint64(val[4])<<24 | uint64(val[5])<<16 |
			uint64(val[6])<<8 | uint64(val[7])
	}
}

// FilterTopicsCommand subscribes to specific APNS topics.
type FilterTopicsCommand struct {
	Token  []byte
	Topics [][]byte // SHA1 hashes of topic strings
}

// ToPayload converts FilterTopicsCommand to binary payload.
func (f *FilterTopicsCommand) ToPayload() *Payload {
	fields := []Field{
		{ID: 1, Value: f.Token},
	}
	for _, topic := range f.Topics {
		fields = append(fields, Field{ID: 2, Value: topic})
	}
	return &Payload{
		ID:     CommandFilterTopics,
		Fields: fields,
	}
}

// SetStateCommand sets connection state.
type SetStateCommand struct {
	State    uint8
	FieldTwo uint32
}

// ToPayload converts SetStateCommand to binary payload.
func (s *SetStateCommand) ToPayload() *Payload {
	fieldTwo := make([]byte, 4)
	fieldTwo[0] = byte(s.FieldTwo >> 24)
	fieldTwo[1] = byte(s.FieldTwo >> 16)
	fieldTwo[2] = byte(s.FieldTwo >> 8)
	fieldTwo[3] = byte(s.FieldTwo)

	return &Payload{
		ID: CommandSetState,
		Fields: []Field{
			{ID: 1, Value: []byte{s.State}},
			{ID: 2, Value: fieldTwo},
		},
	}
}

// IncomingSendMessageCommand is received when a message arrives.
type IncomingSendMessageCommand struct {
	MessageID  []byte
	Token      []byte
	Topic      []byte
	Payload    []byte
	Expiration []byte
	Timestamp  []byte
	Unknown7   []byte
}

// FromPayload parses IncomingSendMessageCommand from payload.
func (i *IncomingSendMessageCommand) FromPayload(p *Payload) {
	i.Token = p.FindField(1)
	i.Topic = p.FindField(2)
	i.Payload = p.FindField(3)
	i.MessageID = p.FindField(4)
	i.Expiration = p.FindField(5)
	i.Timestamp = p.FindField(6)
	i.Unknown7 = p.FindField(7)
}

// KeepAliveCommand is sent/received to maintain connection.
type KeepAliveCommand struct{}

// ToPayload converts KeepAliveCommand to binary payload.
func (k *KeepAliveCommand) ToPayload() *Payload {
	return &Payload{
		ID:     CommandKeepAlive,
		Fields: []Field{},
	}
}
