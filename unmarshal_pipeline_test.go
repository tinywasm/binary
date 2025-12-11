package gobin

import (
	"bytes"
	"reflect"
	"testing"
)

// TestUnmarshalPipeline verifies the complete unmarshal pipeline for struct pointers
// This test covers the step-by-step process that unmarshal follows internally
func TestUnmarshalPipeline(t *testing.T) {
	type InnerStruct struct {
		V int
		S string
	}
	type OuterStruct struct {
		Inner *InnerStruct
		Name  string
	}

	t.Run("CompleteUnmarshalFlow", func(t *testing.T) {
		tb := New()
		// Test data with non-nil pointer
		original := &OuterStruct{
			Inner: &InnerStruct{V: 42, S: "test"},
			Name:  "outer",
		}

		// Step 1: Marshal
		payload, err := tb.Encode(original)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		if len(payload) == 0 {
			t.Fatal("Marshal produced empty payload")
		}

		// Step 2: Verify unmarshal pipeline components
		decoded := &OuterStruct{}
		decoder := NewDecoder(bytes.NewReader(payload))

		// Step 3: Test ValueOf and Indirect operations (core of unmarshal)
		rv := reflect.ValueOf(decoded)
		if rv.Type() == nil {
			t.Fatal("ValueOf returned value with nil type")
		}
		if rv.Kind() != reflect.Ptr {
			t.Errorf("Expected pointer to struct, got kind %v", rv.Kind())
		}

		indirect := reflect.Indirect(rv)
		if indirect.Type() == nil {
			t.Fatal("Indirect returned value with nil type")
		}
		if indirect.Kind() != reflect.Struct {
			t.Errorf("Expected struct after Indirect, got kind %v", indirect.Kind())
		}

		// Step 4: Test codec resolution (scanType functionality)
		typ := indirect.Type()
		structCodec, err := scanType(typ)
		if err != nil {
			t.Fatalf("scanType failed: %v", err)
		}
		if structCodec == nil {
			t.Fatal("scanType returned nil codec")
		}

		// Step 5: Test actual decoding (this was failing before the fix)
		err = structCodec.DecodeTo(decoder, indirect)
		if err != nil {
			t.Fatalf("DecodeTo failed: %v", err)
		}

		// Step 6: Verify the results
		if decoded.Inner == nil {
			t.Error("Decoded inner struct is nil")
		} else {
			if decoded.Inner.V != original.Inner.V {
				t.Errorf("Expected V=%d, got V=%d", original.Inner.V, decoded.Inner.V)
			}
			if decoded.Inner.S != original.Inner.S {
				t.Errorf("Expected S=%s, got S=%s", original.Inner.S, decoded.Inner.S)
			}
		}
		if decoded.Name != original.Name {
			t.Errorf("Expected Name=%s, got Name=%s", original.Name, decoded.Name)
		}
	})

	t.Run("NilPointerHandling", func(t *testing.T) {
		tb := New()
		// Test data with nil pointer
		original := &OuterStruct{
			Inner: nil,
			Name:  "outer_nil",
		}

		// Full roundtrip test
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
		if decoded.Inner != nil {
			t.Error("Expected nil pointer to be preserved")
		}
		if decoded.Name != original.Name {
			t.Errorf("Expected Name=%s, got Name=%s", original.Name, decoded.Name)
		}
	})
}
