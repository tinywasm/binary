package gobin

import (
	"reflect"
	"testing"
)

func TestScanTypeStructFields(t *testing.T) {
	type testStruct struct {
		Name      string
		Timestamp int64
		Payload   []byte
		Ssid      []uint32
	}

	s := &testStruct{}
	rv := reflect.Indirect(reflect.ValueOf(s))
	typ := rv.Type()

	if typ == nil {
		t.Fatal("typ is nil")
	}

	// Test scanType for the struct itself
	t.Logf("Testing scanType for struct type")
	codec, err := scanType(typ)
	if err != nil {
		t.Fatalf("scanType failed for struct: %v", err)
	}

	// Verify we get a struct codec
	if structCodec, ok := codec.(*reflectStructCodec); ok {
		t.Logf("Struct codec has %d field codecs", len(*structCodec))
	} else {
		t.Errorf("Expected *reflectStructCodec, got %T", codec)
	}

	// Test each field type individually to ensure all field types are supported
	numFields := typ.NumField()

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)

		fieldName := field.Name
		fieldTyp := field.Type

		t.Logf("Testing Field %d: %s (Type: %v)", i, fieldName, fieldTyp.Kind())

		// This tests the scanType function for different field types
		fieldCodec, err := scanType(fieldTyp)
		if err != nil {
			t.Fatalf("scanType failed for field %s: %v", fieldName, err)
		}

		// Just verify we got a non-nil codec
		if fieldCodec == nil {
			t.Errorf("Field %d (%s): got nil codec", i, fieldName)
		} else {
			t.Logf("Field %d (%s) codec: %T", i, fieldName, fieldCodec)
		}
	}

	t.Logf("All %d fields processed successfully", numFields)
}
