# Performance Considerations

- **Instance-Based Pooling**: Each GoBin instance maintains its own encoder and decoder pools for optimal resource management
- **Zero Allocations**: Where possible, operations avoid heap allocations for maximum performance
- **Variable-Length Integers**: Integers are encoded with minimal bytes using efficient algorithms
- **Unsafe Operations**: String/byte conversions use unsafe operations for performance when appropriate
- **Slice-Based Caching**: TinyGo-compatible slice-based schema cache provides fast lookups with minimal memory overhead
- **Complete Isolation**: Multiple instances can operate concurrently without contention, improving scalability in multi-goroutine environments