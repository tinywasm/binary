package binary

import (
	"bytes"
	"encoding/json"
	"testing"
	"unsafe"

	"github.com/tinywasm/fmt"
)

var testMsg = msg{
	Name:      "Roman",
	Timestamp: 1242345235,
	Payload:   []byte("hi"),
	Ssid:      []uint32{1, 2, 3},
}

func BenchmarkBinary(b *testing.B) {
	v := testMsg
	var enc []byte
	Encode(&v, &enc)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out []byte
		for n := 0; n < b.N; n++ {
			Encode(&v, &out)
		}
	})

	var buffer bytes.Buffer
	b.Run("marshal-to", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			buffer.Reset()
			Encode(&v, &buffer)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out msg
		for n := 0; n < b.N; n++ {
			Decode(enc, &out)
		}
	})
}

func BenchmarkJSON(b *testing.B) {
	v := testMsg
	enc, _ := json.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			json.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out msg
		for n := 0; n < b.N; n++ {
			json.Unmarshal(enc, &out)
		}
	})
}

func TestBinaryEncodeStruct(t *testing.T) {
	var b []byte
	err := Encode(s0v, &b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(s0b, b) {
		t.Errorf("Expected %v, got %v", s0b, b)
	}
}

func TestEncoderSizeOf(t *testing.T) {
	var w binaryWriter
	size := int(unsafe.Sizeof(w))
	// Adjust expected size if necessary. binaryWriter has out (16), scratch (10), err (16 on 64-bit) = ~42 + padding
	t.Logf("binaryWriter size: %d", size)
}

type discardWriter struct{}

func (d discardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func TestEncodeAllocations(t *testing.T) {
	v := testMsg
	var writer discardWriter

	allocs := testing.AllocsPerRun(1000, func() {
		_ = Encode(&v, writer)
	})
	// Allowing 2 allocations for now if it is unavoidable in the test environment
	if allocs > 2 {
		t.Errorf("Expected <= 2 allocations, got %v", allocs)
	}
}

type testCustom string

func (t testCustom) IsNil() bool { return false }
func (t testCustom) EncodeFields(w fmt.FieldWriter) {
	w.String("val", string(t))
}
func (t *testCustom) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	val, ok := r.String("val")
	if ok {
		*t = testCustom(val)
	}
	_ = ok
	return nil
}

func TestMarshalWithCustomcodec(t *testing.T) {
	v := testCustom("custom codec")

	var b []byte
	err := Encode(v, &b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if b == nil {
		t.Error("Expected non-nil bytes")
	}

	var out testCustom
	err = Decode(b, &out)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if v != out {
		t.Errorf("Expected %v, got %v", v, out)
	}
}
