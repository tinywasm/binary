package binary

import "github.com/tinywasm/fmt"

// Message is the standard inter-module communication envelope.
// All pub/sub messages are encoded as Message before transmission.
type Message struct {
	Topic   string          // routing key: "users.created", "auth.logout"
	Type    fmt.MessageType // Use fmt.MessageType instead of local byte
	ID      uint32          // correlation ID for request/response pairs
	Payload []byte          // binary-encoded body (domain-specific struct)
}

// EncodeFields implements fmt.Encodable
func (m *Message) EncodeFields(w fmt.FieldWriter) {
	w.String("Topic", m.Topic)
	w.Int("Type", int64(m.Type))
	w.Uint("ID", uint64(m.ID))
	w.Bytes("Payload", m.Payload)
}

// DecodeFields implements fmt.Decodable
func (m *Message) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	if m.Topic, ok = r.String("Topic"); !ok {
		return Errorf("missing Topic")
	}
	t, ok := r.Int("Type")
	if !ok {
		return Errorf("missing Type")
	}
	m.Type = fmt.MessageType(t)
	id, ok := r.Uint("ID")
	if !ok {
		return Errorf("missing ID")
	}
	m.ID = uint32(id)
	if m.Payload, ok = r.Bytes("Payload"); !ok {
		return Errorf("missing Payload")
	}
	return nil
}

// IsNil implements fmt.Encodable and fmt.Decodable
func (m *Message) IsNil() bool {
	return m == nil
}
