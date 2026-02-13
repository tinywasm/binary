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
