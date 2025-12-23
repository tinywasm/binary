package binary

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
	if structcodec, ok := codec.(*reflectStructcodec); ok {
		t.Logf("Struct codec has %d field codecs", len(*structcodec))
	} else {
		t.Errorf("Expected *reflectStructcodec, got %T", codec)
	}

	// Test each field type individually to ensure all field types are supported
	numFields := typ.NumField()

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)

		fieldName := field.Name
		fieldTyp := field.Type

		t.Logf("Testing Field %d: %s (Type: %v)", i, fieldName, fieldTyp.Kind())

		// This tests the scanType function for different field types
		fieldcodec, err := scanType(fieldTyp)
		if err != nil {
			t.Fatalf("scanType failed for field %s: %v", fieldName, err)
		}

		// Just verify we got a non-nil codec
		if fieldcodec == nil {
			t.Errorf("Field %d (%s): got nil codec", i, fieldName)
		} else {
			t.Logf("Field %d (%s) codec: %T", i, fieldName, fieldcodec)
		}
	}

	t.Logf("All %d fields processed successfully", numFields)
}
