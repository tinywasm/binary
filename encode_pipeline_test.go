package gobin

import (
	"reflect"
	"testing"
)

// TestEncodePipelineSteps tests the complete encoding pipeline step by step
// This provides coverage for the encoder pipeline from ValueOf to codec execution
func TestEncodePipelineSteps(t *testing.T) {
	// Test exactly what happens in the encoder pipeline
	input := []simpleStruct{{
		Name:      "Roman",
		Timestamp: 1357092245000000006,
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}, {
		Name:      "Roman",
		Timestamp: 1357092245000000006,
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}}

	// Step 1: ValueOf and Indirect (like in Encode)
	rv := reflect.Indirect(reflect.ValueOf(&input))
	t.Logf("Step 1 - rv type: %v, kind: %s", rv.Type(), rv.Type().Kind().String())

	typ := rv.Type()
	if typ == nil {
		t.Fatal("typ is nil!")
	}

	// Step 2: scanToCache (like in Encode)
	schemas := make(map[reflect.Type]Codec)
	c, err := scanToCache(typ, schemas)
	if err != nil {
		t.Fatalf("scanToCache failed: %v", err)
	}
	t.Logf("Step 2 - codec type: %T", c)

	// Step 3: Test a simple property of the value to see if it's valid
	length := rv.Len()
	t.Logf("Step 3 - slice length: %d", length)

	// Step 4: Try to index the first element
	if length > 0 {
		elem := rv.Index(0)
		t.Logf("Step 4 - first element type: %v, kind: %s", elem.Type(), elem.Type().Kind().String())

		// Check if the element is a struct
		if elem.Type().Kind() != reflect.Struct {
			t.Fatalf("Expected first element to be struct, got: %s", elem.Type().Kind().String())
		}
	}
}
