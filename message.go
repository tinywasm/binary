package binary

import "github.com/tinywasm/model"

import "github.com/tinywasm/fmt"

// Message is the standard inter-module communication envelope.
// All pub/sub messages are encoded as Message before transmission.
type Message struct {
	Topic   string          // routing key: "users.created", "auth.logout"
	Type    fmt.MessageType // Use fmt.MessageType instead of local byte
	ID      uint32          // correlation ID for request/response pairs
	Payload []byte          // binary-encoded body (domain-specific struct)
}

// EncodeFields implements model.Encodable
func (m *Message) EncodeFields(w model.FieldWriter) {
	w.String("Topic", m.Topic)
	w.Int("Type", int64(m.Type))
	w.Int("ID", int64(m.ID))
	w.Bytes("Payload", m.Payload)
}

// DecodeFields implements model.Decodable
func (m *Message) DecodeFields(r model.FieldReader) {
	if v, ok := r.String("Topic"); ok {
		m.Topic = v
	}
	if t, ok := r.Int("Type"); ok {
		m.Type = fmt.MessageType(t)
	}
	if id, ok := r.Int("ID"); ok {
		m.ID = uint32(id)
	}
	if v, ok := r.Bytes("Payload"); ok {
		m.Payload = v
	}
}

// IsNil implements model.Encodable and model.Decodable
func (m *Message) IsNil() bool {
	return m == nil
}
