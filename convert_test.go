package gobin

import (
	"bytes"
	"testing"
)

func TestConvert_String(t *testing.T) {
	v := "hi there"

	b := ToBytes(v)
	if len(b) == 0 {
		t.Error("Expected non-empty bytes")
	}
	if v != string(b) {
		t.Errorf("Expected %q, got %q", v, string(b))
	}

	o := ToString(&b)
	if len(b) == 0 {
		t.Error("Expected non-empty bytes")
	}
	if v != o {
		t.Errorf("Expected %q, got %q", v, o)
	}
}

func TestConvert_Bools(t *testing.T) {
	v := []bool{true, false, true, true, false, false}

	b := boolsToBinary(&v)
	if len(b) == 0 {
		t.Error("Expected non-empty bytes")
	}
	expected := []byte{0x1, 0x0, 0x1, 0x1, 0x0, 0x0}
	if !bytes.Equal(expected, b) {
		t.Errorf("Expected %v, got %v", expected, b)
	}

	o := binaryToBools(&b)
	if len(b) == 0 {
		t.Error("Expected non-empty bytes")
	}
	if !equalBoolSlices(v, o) {
		t.Errorf("Expected %v, got %v", v, o)
	}
}

// Helper function to compare bool slices
func equalBoolSlices(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
