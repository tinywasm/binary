package gobin

import (
	"reflect"
	"testing"
)

func TestDecodeFlow(t *testing.T) {
	// Test the complete encode/decode flow with validation
	original := &simpleStruct{
		Name:      "Roman",
		Timestamp: 1357092245000000006, // Unix timestamp in nanoseconds
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}

	// Encode
	tb := New()
	b, err := tb.Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	t.Logf("Encode succeeded, bytes length: %d", len(b))

	decoded := &simpleStruct{}

	// Test the decode prerequisites
	rv := reflect.Indirect(reflect.ValueOf(decoded))
	canAddr := rv.CanAddr()
	if !canAddr {
		t.Fatal("Cannot address - this indicates a fundamental issue with ValueOf/Indirect")
	}

	typ := rv.Type()
	if typ == nil {
		t.Fatal("Type is nil - this indicates a Value creation issue")
	}
	t.Logf("Type: %v, Kind: %v", typ, typ.Kind())

	// Test scanType directly to ensure codec creation works
	codec, err := scanType(typ)
	if err != nil {
		t.Fatalf("scanType failed: %v", err)
	}
	t.Logf("scanType succeeded: %T", codec)

	// Now test the actual decode
	err = tb.Decode(b, decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Validate the results
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, original.Timestamp)
	}
	if string(decoded.Payload) != string(original.Payload) {
		t.Errorf("Payload mismatch: got %q, want %q", decoded.Payload, original.Payload)
	}
	if len(decoded.Ssid) != len(original.Ssid) {
		t.Errorf("Ssid length mismatch: got %d, want %d", len(decoded.Ssid), len(original.Ssid))
	} else {
		for i, v := range original.Ssid {
			if decoded.Ssid[i] != v {
				t.Errorf("Ssid[%d] mismatch: got %d, want %d", i, decoded.Ssid[i], v)
			}
		}
	}

	t.Logf("Decode flow test completed successfully!")
}
