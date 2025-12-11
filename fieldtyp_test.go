package gobin

import (
	"reflect"
	"testing"
)

func TestFieldTypNil(t *testing.T) {
	// Test the exact struct from the failing test
	type simpleStruct struct {
		Name      string
		Timestamp int64
		Payload   []byte
		Ssid      []uint32
	}

	s := &simpleStruct{}
	rv := reflect.Indirect(reflect.ValueOf(s))
	typ := rv.Type()

	if typ == nil {
		t.Fatal("rv.Type() returned nil")
	}

	// Check each field individually
	numFields := typ.NumField()

	t.Logf("Struct has %d fields", numFields)

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)

		t.Logf("Field %d: Name=%s, Typ=%v", i, field.Name, field.Type)

		if field.Type == nil {
			t.Errorf("❌ Field %d (%s) has nil Typ!", i, field.Name)
		} else {
			t.Logf("✅ Field %d (%s) has Typ: %v, Kind: %v", i, field.Name, field.Type, field.Type.Kind())

			// Test scanType on this field
			codec, err := scanType(field.Type)
			if err != nil {
				t.Errorf("❌ scanType failed for field %d (%s): %v", i, field.Name, err)
			} else {
				t.Logf("✅ scanType succeeded for field %d (%s): %T", i, field.Name, codec)
			}
		}
	}
}
