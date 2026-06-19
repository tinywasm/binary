package binary

import (
	"bytes"
	"io"
	"testing"

	"github.com/tinywasm/fmt"
)

func TestCoverageGaps(t *testing.T) {
	// SetLog
	SetLog(func(msg ...any) {})
	SetLog(nil)

	// Encode/Decode errors and branches
	t.Run("EncodeDecodeBranches", func(t *testing.T) {
		// Encode to invalid output
		err := Encode(&simpleStruct{Name: "test"}, 1)
		if err == nil {
			t.Error("Expected error encoding to int")
		}

		// Decode from invalid input
		var bs simpleStruct
		err = Decode(1, &bs)
		if err == nil {
			t.Error("Expected error decoding from int")
		}

		// Decode from io.Reader
		var out testDecodableString
		err = Decode(bytes.NewReader([]byte{4, 't', 'e', 's', 't'}), &out)
		if err != nil {
			t.Errorf("Unexpected error decoding from reader: %v", err)
		}
		if out.val != "test" {
			t.Errorf("Expected 'test', got %v", out.val)
		}

		// Encode to io.Writer
		var buf bytes.Buffer
		err = Encode(testEncodableString("test"), &buf)
		if err != nil {
			t.Errorf("Unexpected error encoding to writer: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), []byte{4, 't', 'e', 's', 't'}) {
			t.Errorf("Expected encoded 'test', got %v", buf.Bytes())
		}
	})

	t.Run("CodecCoverage", func(t *testing.T) {
		// boolSliceCodec replacement test
		type BoolSlice struct {
			B []bool
		}
		bs := &BoolSlice{B: []bool{true, false, true}}
		// Manually implementing Encodable/Decodable for the test
		encBs := &testEncodableBoolSlice{B: bs.B}
		var data []byte
		err := Encode(encBs, &data)
		if err != nil {
			t.Errorf("Encode boolSlice failed: %v", err)
		}
		decBs := &testEncodableBoolSlice{}
		err = Decode(data, decBs)
		if err != nil {
			t.Errorf("Decode boolSlice failed: %v", err)
		}
	})

	t.Run("ReaderCoverage", func(t *testing.T) {
		r := newSliceReader([]byte{1, 2, 3})
		if r.Size() != 3 {
			t.Errorf("Expected size 3, got %d", r.Size())
		}
		// Reach EOF
		r.Slice(3)
		if r.Len() != 0 {
			t.Errorf("Expected len 0, got %d", r.Len())
		}

		_, err := r.ReadByte()
		if err != io.EOF {
			t.Errorf("Expected EOF, got %v", err)
		}

		_, err = r.Read(make([]byte, 1))
		if err != io.EOF {
			t.Errorf("Expected EOF, got %v", err)
		}

		_, err = r.Slice(1)
		if err != io.EOF {
			t.Errorf("Expected EOF, got %v", err)
		}

		// ReadUvarint overflow/EOF
		r2 := newSliceReader([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01})
		_, err = r2.ReadUvarint()
		if err == nil {
			t.Error("Expected error on varint overflow")
		}

		r3 := newSliceReader([]byte{0x80})
		_, err = r3.ReadUvarint()
		if err != io.EOF {
			t.Errorf("Expected EOF, got %v", err)
		}
	})
}

type testEncodableBoolSlice struct {
	B []bool
}

func (t *testEncodableBoolSlice) IsNil() bool { return t == nil }
func (t *testEncodableBoolSlice) EncodeFields(w fmt.FieldWriter) {
	aw := w.Array("B", len(t.B))
	for i := 0; i < len(t.B); i++ {
		aw.Bool(t.B[i])
	}
}
func (t *testEncodableBoolSlice) DecodeFields(r fmt.FieldReader) error {
	if ar, ok := r.Array("B"); ok {
		t.B = make([]bool, ar.Len())
		for i := 0; i < ar.Len(); i++ {
			t.B[i] = ar.Bool(i)
		}
	}
	return nil
}
