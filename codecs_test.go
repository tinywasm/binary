package binary

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/tinywasm/fmt"
)

// Message represents a message to be flushed
type msg struct {
	Name      string
	Timestamp int64
	Payload   []byte
	Ssid      []uint32
}

func (m *msg) IsNil() bool { return m == nil }

func (m *msg) EncodeFields(w fmt.FieldWriter) {
	w.String("Name", m.Name)
	w.Int("Timestamp", m.Timestamp)
	w.Bytes("Payload", m.Payload)
	aw := w.Array("Ssid", len(m.Ssid))
	for i := 0; i < len(m.Ssid); i++ {
		aw.Int(int64(m.Ssid[i]))
	}
}

func (m *msg) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	m.Name, ok = r.String("Name")
	t, ok := r.Int("Timestamp")
	m.Timestamp = t
	m.Payload, ok = r.Bytes("Payload")
	if ar, ok := r.Array("Ssid"); ok {
		m.Ssid = make([]uint32, ar.Len())
		for i := 0; i < ar.Len(); i++ {
			m.Ssid[i] = uint32(ar.Int(i))
		}
	}
	_ = ok
	return nil
}

type s0 struct {
	A string
	B string
	C int16
}

func (s *s0) IsNil() bool { return s == nil }

func (s *s0) EncodeFields(w fmt.FieldWriter) {
	w.String("A", s.A)
	w.String("B", s.B)
	w.Int("C", int64(s.C))
}

func (s *s0) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	s.A, ok = r.String("A")
	s.B, ok = r.String("B")
	v, ok := r.Int("C")
	s.C = int16(v)
	_ = ok
	return nil
}

var (
	s0v = &s0{"A", "B", 1}
	s0b = []byte{0x1, 0x41, 0x1, 0x42, 0x2}
)

type simpleStruct struct {
	Name      string
	Timestamp int64
	Payload   []byte
	Ssid      []uint32
}

func (s *simpleStruct) IsNil() bool { return s == nil }

func (s *simpleStruct) EncodeFields(w fmt.FieldWriter) {
	w.String("Name", s.Name)
	w.Int("Timestamp", s.Timestamp)
	w.Bytes("Payload", s.Payload)
	aw := w.Array("Ssid", len(s.Ssid))
	for i := 0; i < len(s.Ssid); i++ {
		aw.Int(int64(s.Ssid[i]))
	}
}

func (s *simpleStruct) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	if s.Name, ok = r.String("Name"); !ok {
		return Errorf("missing Name")
	}
	if s.Timestamp, ok = r.Int("Timestamp"); !ok {
		return Errorf("missing Timestamp")
	}
	if s.Payload, ok = r.Bytes("Payload"); !ok {
		return Errorf("missing Payload")
	}
	if ar, ok := r.Array("Ssid"); ok {
		s.Ssid = make([]uint32, ar.Len())
		for i := 0; i < ar.Len(); i++ {
			s.Ssid[i] = uint32(ar.Int(i))
		}
	}
	return nil
}

type sliceStruct struct {
	Payload []byte
}

func (s *sliceStruct) IsNil() bool { return s == nil }

func (s *sliceStruct) EncodeFields(w fmt.FieldWriter) {
	w.Bytes("Payload", s.Payload)
}

func (s *sliceStruct) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	s.Payload, ok = r.Bytes("Payload")
	_ = ok
	return nil
}

func TestBinaryEncode_EOF(t *testing.T) {
	v := &sliceStruct{
		Payload: nil,
	}
	output := []byte{0x0}

	var b []byte; err := Encode(v, &b)
	assertNoError(t, err)
	assertEqualBytes(t, output, b)

	s := &sliceStruct{}
	err = Decode(b, s)
	assertNoError(t, err)
	assertEqual(t, v, s)
}

func TestBinaryEncodeSimpleStruct(t *testing.T) {
	v := &simpleStruct{
		Name:      "Roman",
		Timestamp: 1357092245000000006, // Unix timestamp in nanoseconds
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}

	var b []byte; err := Encode(v, &b)
	assertNoError(t, err)
	// For now, let's see what actual output we get
	t.Logf("Actual output: %v", b)

	s := &simpleStruct{}
	err = Decode(b, s)
	assertNoError(t, err)
	assertEqual(t, v, s)
}

type s2 struct {
	b []byte
}

func (s *s2) IsNil() bool { return s == nil }

func (s *s2) EncodeFields(w fmt.FieldWriter) {
	w.Bytes("b", s.b)
}

func (s *s2) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	s.b, ok = r.Bytes("b")
	if !ok {
		return Errorf("missing b")
	}
	return nil
}

func TestBinaryMarshalUnMarshaler(t *testing.T) {
	s2v := &s2{[]byte{0x13}}
	var b []byte; err := Encode(s2v, &b)
	assertNoError(t, err)
	assertEqualBytes(t, []byte{0x1, 0x13}, b)
}

type encodableUint64 uint64
func (u encodableUint64) IsNil() bool { return false }
func (u encodableUint64) EncodeFields(w fmt.FieldWriter) {
	w.Uint("val", uint64(u))
}

type decodableUint64 struct { val uint64 }
func (u *decodableUint64) IsNil() bool { return u == nil }
func (u *decodableUint64) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	u.val, ok = r.Uint("val")
	_ = ok
	return nil
}

func TestMarshalUnMarshalTypeAliases(t *testing.T) {
	f := encodableUint64(32)
	var b []byte; err := Encode(f, &b)
	assertNoError(t, err)
	assertEqual(t, []byte{0x20}, b)
}

type T1 struct {
	ID    uint64
	Name  string
	Slice []int
}

func (t *T1) IsNil() bool { return t == nil }

func (t *T1) EncodeFields(w fmt.FieldWriter) {
	w.Uint("ID", t.ID)
	w.String("Name", t.Name)
	aw := w.Array("Slice", len(t.Slice))
	for i := 0; i < len(t.Slice); i++ {
		aw.Int(int64(t.Slice[i]))
	}
}

func (t *T1) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	t.ID, ok = r.Uint("ID")
	t.Name, ok = r.String("Name")
	if ar, ok := r.Array("Slice"); ok {
		t.Slice = make([]int, ar.Len())
		for i := 0; i < ar.Len(); i++ {
			t.Slice[i] = int(ar.Int(i))
		}
	}
	_ = ok
	return nil
}

type StructWithT1 struct {
	V1 T1
	V2 uint64
	V3 T1
}

func (s *StructWithT1) IsNil() bool { return s == nil }

func (s *StructWithT1) EncodeFields(w fmt.FieldWriter) {
	w.Object("V1", &s.V1)
	w.Uint("V2", s.V2)
	w.Object("V3", &s.V3)
}

func (s *StructWithT1) DecodeFields(r fmt.FieldReader) error {
	r.Object("V1", &s.V1)
	var ok bool
	s.V2, ok = r.Uint("V2")
	r.Object("V3", &s.V3)
	_ = ok
	return nil
}

func TestStructWithStruct(t *testing.T) {
	s := StructWithT1{V1: T1{1, "1", []int{1}}, V2: 2, V3: T1{3, "3", []int{3}}}
	
	var data []byte; err := Encode(&s, &data)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	v := StructWithT1{}
	err = Decode(data, &v)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	if !reflect.DeepEqual(s, v) {
		t.Fatalf("got= %#v\nwant=%#v\n", v, s)
	}
}

type testEncodableString string

func (s testEncodableString) IsNil() bool { return false }
func (s testEncodableString) EncodeFields(w fmt.FieldWriter) {
	w.String("val", string(s))
}

type testDecodableString struct {
	val string
}

func (s *testDecodableString) IsNil() bool { return s == nil }
func (s *testDecodableString) DecodeFields(r fmt.FieldReader) error {
	var ok bool
	s.val, ok = r.String("val")
	if !ok {
		return Errorf("missing val")
	}
	return nil
}

// Helper functions for testing
func assertEqual(t *testing.T, expected, actual any) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func assertEqualBytes(t *testing.T, expected, actual []byte) {
	if !bytes.Equal(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func assertEqualInt(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}
