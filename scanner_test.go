package binary

import (
	"bytes"
	"reflect"
	"testing"
)

type testCustom string

// GetBinarycodec retrieves a custom binary codec.
func (s *testCustom) GetBinarycodec() codec {
	return new(stringcodec)
}

func TestScanner(t *testing.T) {
	rt := reflect.Indirect(reflect.ValueOf(s0v)).Type()
	codec, err := scanType(rt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if codec == nil {
		t.Fatal("Expected non-nil codec")
	}

	var b bytes.Buffer
	e := newEncoder(&b)
	err = codec.encodeTo(e, reflect.Indirect(reflect.ValueOf(s0v)))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(s0b, b.Bytes()) {
		t.Errorf("Expected %v, got %v", s0b, b.Bytes())
	}
}

func TestScanner_Custom(t *testing.T) {
	v := testCustom("test")
	rt := reflect.Indirect(reflect.ValueOf(v)).Type()
	codec, err := scanType(rt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if codec == nil {
		t.Fatal("Expected non-nil codec")
	}
}

// TestScannerComposed removed - uses Partition struct with maps which are not supported
// Maps are intentionally not supported in Binary for WebAssembly optimization
/*
func TestScannerComposed(t *testing.T) {
	codec, err := scanType(reflect.TypeOf(Partition{}))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if codec == nil {
		t.Fatal("Expected non-nil codec")
	}
}

type Partition struct {
	Strings
	Filters map[uint32][]uint64
}

type Strings struct {
	Key  string
	Fill []uint64
	Hash []uint32
	Data map[uint64][]byte
}
*/
