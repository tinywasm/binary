# TinyBin Agents Guidelines

TinyBin is a high-performance binary serialization library for Go, optimized for TinyGo and resource-constrained environments.

## Core Restrictions

- **No `reflect`**: Do not use the `reflect` package in the core library. All serialization must be handled through the `fmt.Encodable` and `fmt.Decodable` interfaces.
- **No `sync`**: Avoid using the `sync` package. The library should be 0-alloc and map-free, eliminating the need for complex synchronization or caching.
- **No `map`**: Do not use the `map` type in the serialization path.
- **No `stdlib`**: Minimize dependencies on the Go standard library, especially those that are large or not well-supported in TinyGo.
- **WASM+Backend Agnostic**: The code must compile and run correctly on both WASM (TinyGo) and standard Go backends.
- **0-alloc**: The `Encode` process should aim for zero allocations.

## Testing and Verification

- **`gotest`**: Use the `gotest` command to run tests. Do not use `go test` directly.
- **Performance**: Always verify that changes do not introduce performance regressions or new allocations. Update `docs/BENCHMARK.md` when performance characteristics change.

## Codec Contract

The library follows the codec contract defined in `github.com/tinywasm/fmt`.

- `EncodeFields(w fmt.FieldWriter)`: Used for serializing objects.
- `DecodeFields(r fmt.FieldReader) error`: Used for deserializing objects.

In the binary format, field names are omitted for compactness. The order of fields in `EncodeFields` must match the order in `DecodeFields`.
