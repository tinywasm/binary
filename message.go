package binary

// Message is the standard inter-module communication envelope.
// All pub/sub messages are encoded as Message before transmission.
type Message struct {
	Topic   string // routing key: "users.created", "auth.logout"
	Type    uint8  // 0=event, 1=request, 2=response, 3=error
	ID      uint32 // correlation ID for request/response pairs
	Payload []byte // binary-encoded body (domain-specific struct)
}

// MessageType constants
const (
	MsgTypeEvent    uint8 = 0
	MsgTypeRequest  uint8 = 1
	MsgTypeResponse uint8 = 2
	MsgTypeError    uint8 = 3
)
