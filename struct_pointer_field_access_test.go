package binary

import (
	"reflect"
	"testing"
)

// TestStructPointerFieldAccess verifies that struct fields containing pointers
// can be properly accessed and marshaled/unmarshaled through reflect
func TestStructPointerFieldAccess(t *testing.T) {
	type InnerStruct struct {
		V int
	}
	type OuterStruct struct {
		Ptr *InnerStruct
	}

	// Test case 1: Non-nil pointer
	t.Run("NonNilPointer", func(t *testing.T) {
		original := &OuterStruct{Ptr: &InnerStruct{V: 42}}

		// Verify basic field access works correctly
		rv := reflect.ValueOf(original)
		elem := rv.Elem()

		// Verify we can access the pointer field
		ptrField := elem.Field(0)

		// Verify the field has correct type and kind
		if ptrField.Type() == nil {
			t.Fatal("ptrField.Type() returned nil")
		}
		if ptrField.Kind() != reflect.Ptr {
			t.Errorf("Expected pointer kind, got %v", ptrField.Kind())
		}

		// Verify marshal/unmarshal roundtrip
		var payload []byte
		err := Encode(original, &payload)
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		decoded := &OuterStruct{}
		err = Decode(payload, decoded)
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		// Verify the result
		if decoded.Ptr == nil {
			t.Fatal("Decoded pointer is nil")
		}
		if decoded.Ptr.V != original.Ptr.V {
			t.Errorf("Expected V=%d, got V=%d", original.Ptr.V, decoded.Ptr.V)
		}
	})

	// Test case 2: Nil pointer
	t.Run("NilPointer", func(t *testing.T) {
		original := &OuterStruct{Ptr: nil}

		var payload []byte
		err := Encode(original, &payload)
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		decoded := &OuterStruct{}
		err = Decode(payload, decoded)
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		// Verify nil pointer is preserved
		if decoded.Ptr != nil {
			t.Error("Expected nil pointer, but got non-nil")
		}
	})
}
