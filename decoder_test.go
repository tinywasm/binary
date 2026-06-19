package binary

import (
	"io"
	"reflect"
	"testing"
)

func TestBinaryDecodeStruct(t *testing.T) {
	s := &s0{}
	err := Decode(s0b, s)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(s0v, s) {
		t.Errorf("Expected %v, got %v", s0v, s)
	}
}

type oneByteReader struct {
	content []byte
}

// Read method of io.Reader reads *up to* len(buf) bytes.
// It is possible to read LESS, and it can happen when reading a file.
func (r *oneByteReader) Read(buf []byte) (n int, err error) {
	if len(r.content) == 0 {
		err = io.EOF
		return
	}

	if len(buf) == 0 {
		return
	}
	n = 1
	buf[0] = r.content[0]
	r.content = r.content[1:]
	return
}

func TestDecodeFromReader(t *testing.T) {
	data := testEncodableString("data string")
	var encoded []byte
	err := Encode(data, &encoded)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	decoded := &testDecodableString{}
	err = Decode(&oneByteReader{content: encoded}, decoded)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(string(data), decoded.val) {
		t.Errorf("Expected %v, got %v", string(data), decoded.val)
	}
}
