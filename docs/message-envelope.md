# Base 3: tinywasm/binary — Message Envelope

## Change: Add `Message` type

```go
// binary/message.go (new file)
package binary

// Message is the standard inter-module communication envelope.
// All pub/sub messages are encoded as Message before transmission.
type Message struct {
    Topic   string  // routing key: "users.created", "auth.logout"
    Type    uint8   // 0=event, 1=request, 2=response, 3=error
    ID      uint32  // correlation ID for request/response pairs
    Payload []byte  // binary-encoded body (domain-specific struct)
}

// MessageType constants
const (
    MsgTypeEvent    uint8 = 0
    MsgTypeRequest  uint8 = 1
    MsgTypeResponse uint8 = 2
    MsgTypeError    uint8 = 3
)
```

## Why in binary (not bus)?
`Message` is the **wire format** — it must be available on both sides (server = imports bus; modules = imports binary for encoding payloads). Keeping it in `binary` avoids a circular dependency (`bus` → `binary`, never `binary` → `bus`).

## Encoding contract
- `Message` itself is encoded via `binary.Encode(msg, &buf)` — standard usage
- `Payload` inside is the caller's responsibility to encode with `binary.Encode(domainStruct, &msg.Payload)`
