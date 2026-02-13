# Base 3: tinywasm/binary — Message Envelope

## Change: Add `Message` type

```go
// binary/message.go
package binary

import "github.com/tinywasm/fmt"

// Message is the standard inter-module communication envelope.
// All pub/sub messages are encoded as Message before transmission.
type Message struct {
    Topic   string          // routing key: "users.created", "auth.logout"
    Type    fmt.MessageType // Event, Request, Response, Error
    ID      uint32          // correlation ID for request/response pairs
    Payload []byte          // binary-encoded body (domain-specific struct)
}
```

## Why in binary (not bus)?
`Message` is the **wire format** — it must be available on both sides (server = imports bus; modules = imports binary for encoding payloads). Keeping it in `binary` avoids a circular dependency (`bus` → `binary`, never `binary` → `bus`).

## Encoding contract
- `Message` itself is encoded via `binary.Encode(msg, &buf)` — standard usage
- `Payload` inside is the caller's responsibility to encode with `binary.Encode(domainStruct, &msg.Payload)`
