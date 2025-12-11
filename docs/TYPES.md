# Supported Data Types

GoBin automatically handles encoding and decoding for the following types:

## Primitive Types
- `bool` - encoded as a single byte (0 or 1)
- `int`, `int8`, `int16`, `int32`, `int64` - variable-length encoded
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` - variable-length encoded
- `float32`, `float64` - IEEE 754 binary representation
- `string` - UTF-8 bytes with length prefix

## Composite Types
- **Slices** - length-prefixed sequence of elements
  ```go
  []int{1, 2, 3, 4, 5}     // → [5, 1, 2, 3, 4, 5]
  []string{"a", "b", "c"}   // → [3, "a", "b", "c"]
  ```
- **Arrays** - fixed-size sequence of elements
  ```go
  [3]int{1, 2, 3}          // → [1, 2, 3]
  ```
- **Structs** - field-by-field encoding
  ```go
  type Point struct {
      X, Y int
  }
  ```
- **Pointers** - nil check followed by element encoding
  ```go
  var ptr *MyStruct = &MyStruct{...}  // → [0, ...data...]
  var nilPtr *MyStruct = nil          // → [1]
  ```