# Benchmarks

This document details the performance comparisons between `tinywasm/binary` (with and without `namedHandler` optimization) and Go's standard `encoding/json`.

## Performance Summary

Tests were performed using a typical message structure (`testMsg`) containing strings, timestamps, and numeric slices.

| Library | Operation | Speed (ns/op) | Memory (B/op) | Allocations |
| :--- | :--- | :--- | :--- | :--- |
| **JSON (stdlib)** | Marshal | 249.6 ns | 80 B | 1 |
| **JSON (stdlib)** | Unmarshal | 1145.0 ns | 240 B | 7 |
| **Binary (Reflect Only)** | Marshal | 249.7 ns | 112 B | 2 |
| **Binary (Reflect Only)** | Unmarshal | 145.5 ns | 24 B | 1 |
| **Binary (Named)** | Marshal | **125.7 ns** | 112 B | 2 |
| **Binary (Named)** | Unmarshal | **89.45 ns** | 24 B | 1 |

### Key Conclusions

1.  **Named Optimization**: Implementing the `namedHandler` interface (by providing a `HandlerName()`) improves Marshal performance by **2x** and Unmarshal by **1.6x** compared to using pure reflection on every call.
2.  **Vs JSON**: `Binary (Named)` is approximately **2x faster** than JSON for encoding and up to **12x faster** for decoding.
3.  **Memory**: Binary maintains a very low memory profile, with only 1 allocation in Unmarshal vs JSON's 7.

> [!TIP]
> For maximum performance in TinyGo/WASM environments, it is highly recommended that high-traffic structures implement the internal interface to take advantage of name-based caching.
