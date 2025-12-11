package gobin

import (
	"reflect"
	"testing"
)

func TestDecoderFullFlow(t *testing.T) {
	// This replicates the exact flow from decoder.go
	type simpleStruct struct {
		Name      string
		Timestamp int64
		Payload   []byte
		Ssid      []uint32
	}

	s := &simpleStruct{}

	// decoder.go line 46: rv := reflect.Indirect(reflect.ValueOf(v))
	rv := reflect.Indirect(reflect.ValueOf(s))

	// Check CanAddr
	canAddr := rv.CanAddr()
	if !canAddr {
		t.Fatal("rv.CanAddr() returned false")
	}

	// decoder.go line 52: scanToCache(rv.Type(), d.schemas)
	typ := rv.Type()
	if typ == nil {
		t.Fatal("rv.Type() returned nil - this is the problem!")
	}

	t.Logf("rv.Type() = %v, Kind: %v", typ, typ.Kind())

	// Create a mock cache like in decoder
	cache := make(map[reflect.Type]Codec)

	// decoder.go line 52: scanToCache(rv.Type(), d.schemas)
	codec, err := scanToCache(typ, cache)
	if err != nil {
		t.Fatalf("scanToCache failed: %v", err)
	}

	if codec == nil {
		t.Fatal("scanToCache returned nil codec")
	}

	t.Logf("scanToCache succeeded, codec: %T", codec)
}
