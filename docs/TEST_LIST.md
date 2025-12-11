# GoBin Test Coverage Plan

## Objective
Increase test coverage for GoBin serialization by adding missing scenarios and edge cases. Use shared fixtures for **new tests** to avoid code duplication. Existing tests should be preserved and only updated if they are broken or redundant.

## Core Structures (Fixtures)
Use these structures for **new test cases** in `shared_test.go`.

### 1. `FixtureBasic`
Covers all primitive types, standard slices, and basic logic.
```go
type FixtureBasic struct {
    Name      string   // UTF-8, empty, large strings
    Timestamp int64    // UnixNano (Time/Audit fields)
    Payload   []byte   // Binary data
    Tags      []uint32 // Slice of primitives
    Count     int16    // Small integers
    Active    bool     // Booleans
    Score     float64  // Floating point
}
```

### 2. `FixtureComplex`
Covers nesting, pointers, and composition patterns.
```go
type FixtureComplex struct {
    ID        uint64
    Primary   FixtureBasic   // Embedded/Nested struct
    Secondary *FixtureBasic  // Pointer to struct (nil/non-nil)
    List      []FixtureBasic // Slice of structs
    Matrix    [3]int         // Fixed array
}
```

## Detailed Test Cases Checklist

### 1. Primitive & Basic Types (FixtureBasic)
*Target: `FixtureBasic` struct*

- [ ] **TC-001: String Handling**
  - `Name`: "" (Empty string)
  - `Name`: "Hello World" (ASCII)
  - `Name`: "ñandú 🚀" (UTF-8/Emoji)
  - `Name`: String > 1KB (Large string)

- [ ] **TC-002: Timestamp/Int64 Precision**
  - `Timestamp`: `time.Now().UnixNano()` (Current)
  - `Timestamp`: `0` (Zero value/Null equivalent)
  - `Timestamp`: `-1` (Negative/Pre-1970)
  - `Timestamp`: `MaxInt64` (Far future)

- [ ] **TC-003: Binary Data**
  - `Payload`: `nil`
  - `Payload`: `[]byte{}` (Empty slice)
  - `Payload`: `[]byte{0x00, 0xFF, ...}` (Binary content)

- [ ] **TC-004: Slice of Primitives**
  - `Tags`: `nil`
  - `Tags`: `[]uint32{}` (Empty)
  - `Tags`: `[]uint32{1, 2, 3}` (Populated)

### 2. Complex Structures (FixtureComplex)
*Target: `FixtureComplex` struct*

- [ ] **TC-005: Nested Structs**
  - `Primary`: Populated `FixtureBasic` struct.
  - Verify fields inside `Primary` are correctly serialized/deserialized.

- [ ] **TC-006: Pointer Handling (Nil)**
  - `Secondary`: `nil`
  - Verify it decodes back to `nil` (not an empty struct).

- [ ] **TC-007: Pointer Handling (Populated)**
  - `Secondary`: `&FixtureBasic{...}`
  - Verify it decodes back to a valid pointer with correct data.

- [ ] **TC-008: Slice of Structs**
  - `List`: `nil`
  - `List`: `[]FixtureBasic{}` (Empty)
  - `List`: `[]FixtureBasic{{...}, {...}}` (Multiple items)

- [ ] **TC-009: Fixed Arrays**
  - `Matrix`: `[3]int{0, 0, 0}`
  - `Matrix`: `[3]int{1, 2, 3}`

### 3. Edge Cases & Performance

- [ ] **TC-010: Zero Values**
  - Encode a `FixtureComplex` where ALL fields are zero values.
  - Verify decoding results in exact zero-value match.

- [ ] **TC-011: Large Collections**
  - `List`: Slice containing 10,000 `FixtureBasic` items.
  - Verify data integrity and absence of panics.

- [ ] **TC-012: Deep Nesting**
  - (If applicable via recursive definition or chain)
  - Verify stack doesn't overflow on reasonable depth.

## Constraints & Dependencies
- **Forbidden Packages**: Do NOT use Go standard library packages `fmt`, `strconv`, `errors`, or `strings` in the implementation code.
- **Allowed Packages**: Use `tinystring` for string manipulation and formatting.
- **Reasoning**: Maintain minimal binary size and dependency footprint for WebAssembly/TinyGo compatibility.

## Implementation Tasks
1. Create `shared_test.go` with `FixtureBasic` and `FixtureComplex`.
2. Implement **new test cases** (TC-001 to TC-012) using these fixtures.
3. **Optional**: Update existing tests to use shared fixtures ONLY if it simplifies maintenance or fixes bugs. Do not blindly refactor working tests.
4. Verify all tests (new and existing) pass.
