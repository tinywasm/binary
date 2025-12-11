package gobin

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
		tb := New()
		payload, err := tb.Encode(original)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &OuterStruct{}
		err = tb.Decode(payload, decoded)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
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
		tb := New()
		original := &OuterStruct{Ptr: nil}

		payload, err := tb.Encode(original)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		decoded := &OuterStruct{}
		err = tb.Decode(payload, decoded)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		// Verify nil pointer is preserved
		if decoded.Ptr != nil {
			t.Error("Expected nil pointer, but got non-nil")
		}
	})
}
