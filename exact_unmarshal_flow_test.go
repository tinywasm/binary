package gobin

import (
	"reflect"
	"testing"
)

func TestExactUnmarshalFlow(t *testing.T) {
	// Reproduce the exact flow from Unmarshal -> Decode -> scanToCache
	s := &simpleStruct{
		Name:      "Roman",
		Timestamp: 1357092245000000006,
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}

	// Marshal first (this should work)
	tb := New()
	b, err := tb.Encode(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("Marshal succeeded: %v", b)

	// Now test the decode flow step by step
	dest := &simpleStruct{}

	// Step 1: Get the reflect value (decoder.go line 46)
	rv := reflect.Indirect(reflect.ValueOf(dest))
	t.Logf("rv.Type(): %v", rv.Type())

	// Step 2: Check if Type is nil
	if rv.Type() == nil {
		t.Fatal("rv.Type() is nil - this is the problem")
	}

	// Step 3: Call scanToCache directly (decoder.go line 52)
	cache := make(map[reflect.Type]Codec)
	codec, err := scanToCache(rv.Type(), cache)
	if err != nil {
		t.Fatalf("scanToCache failed: %v", err)
	}

	t.Logf("scanToCache succeeded: %T", codec)

	// If this passes, the problem is elsewhere
	t.Logf("Test passed - problem might be in the actual decode flow")
}
