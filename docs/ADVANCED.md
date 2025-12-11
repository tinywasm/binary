# Advanced Usage

## Multiple Instance Usage

```go
// Create multiple isolated instances
httpTB := gobin.New()
grpcTB := gobin.New()
kafkaTB := gobin.New()

// Each instance maintains its own cache and pools
httpData, _ := httpTB.Encode(data)
grpcData, _ := grpcTB.Encode(data)
kafkaData, _ := kafkaTB.Encode(data)
```

## Custom Instance with Logging

```go
// Create instance with custom logging for debugging
tb := gobin.New(func(msg ...any) {
    log.Printf("GoBin Debug: %v", msg)
})

// Use like normal
data, err := tb.Encode(myStruct)
if err != nil {
    log.Printf("Encoding failed: %v", err)
}
```

## Concurrent Usage

```go
tb := gobin.New()

// Safe concurrent usage - internal pooling handles synchronization
go func() {
    data, _ := tb.Encode(data1)
    process(data)
}()

go func() {
    data, _ := tb.Encode(data2)
    process(data)
}()
```

## Error Handling

```go
tb := gobin.New()

data, err := tb.Encode(myValue)
if err != nil {
    // Handle encoding error
    log.Printf("Encoding failed: %v", err)
}

var result MyType
err = tb.Decode(data, &result)
if err != nil {
    // Handle decoding error
    log.Printf("Decoding failed: %v", err)
}
```

## Multiple Instance Patterns

**Microservices Pattern**: Different services can use separate instances for complete isolation.

```go
type ProtocolManager struct {
    httpGoBin  *gobin.GoBin
    grpcGoBin  *gobin.GoBin
    kafkaGoBin *gobin.GoBin
}

func NewProtocolManager() *ProtocolManager {
    return &ProtocolManager{
        httpGoBin:  gobin.New(), // Production: no logging
        grpcGoBin:  gobin.New(),
        kafkaGoBin: gobin.New(),
    }
}
```

**Concurrent Processing**: Multiple instances can be used safely across goroutines.

```go
// Each goroutine gets its own instance for complete isolation
go func() {
    tb := gobin.New()
    data, _ := tb.Encode(data1)
    process(data)
}()

go func() {
    tb := gobin.New()
    data, _ := tb.Encode(data2) // Completely independent
    process(data)
}()
```