package binary

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestIntegrationCoverage(t *testing.T) {
	t.Run("ConvertHelpers", func(t *testing.T) {
		// Test toString
		b := []byte("hello")
		s := toString(&b)
		if s != "hello" {
			t.Errorf("expected hello, got %s", s)
		}

		// Test boolsToBinary and binaryToBools
		bools := []bool{true, false, true}
		bin := boolsToBinary(&bools)
		if !bytes.Equal(bin, []byte{1, 0, 1}) {
			t.Errorf("expected [1 0 1], got %v", bin)
		}

		boolsOut := binaryToBools(&bin)
		if len(boolsOut) != 3 || !boolsOut[0] || boolsOut[1] || !boolsOut[2] {
			t.Errorf("expected [true false true], got %v", boolsOut)
		}
	})

	t.Run("FindSchemaByNameCache", func(t *testing.T) {
		inst := newInstance()
		type namedT struct{ Name string }
		typ := reflect.TypeOf(namedT{})
		codec, _ := scan(typ)

		inst.addSchema(typ, codec, "named")

		// Hit findSchemaByName cache
		c, rt, found := inst.findSchemaByName("named")
		if !found || rt != typ || c != codec {
			t.Errorf("expected found with correct type and codec")
		}
	})

	t.Run("DecoderReadSlice", func(t *testing.T) {
		data := []byte{3, 1, 2, 3}
		d := newDecoder(bytes.NewReader(data))
		// Use a slice reader for the inner reader if possible,
		// but simple reader works too since we just want to hit the method
		b, err := d.readSlice()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(b, []byte{1, 2, 3}) {
			t.Errorf("expected [1 2 3], got %v", b)
		}

		// EOF error branch
		dErr := newDecoder(bytes.NewReader([]byte{10}))
		_, err = dErr.readSlice()
		if err == nil {
			t.Error("expected error for truncated slice")
		}
	})

	t.Run("EncoderErrorBranches", func(t *testing.T) {
		e := newEncoder(io.Discard)
		e.err = io.EOF

		// scanner error branch in encode
		// We need a way to make scanToCache fail
		inst := newInstance()
		e.tb = inst

		val := make(chan int)
		err := e.encode(val)
		if err == nil {
			t.Error("expected error for unsupported type")
		}
	})

	t.Run("MapCodecErrors", func(t *testing.T) {
		// reflect.TypeOf(make(chan int)) is unsupported
		mtyp := reflect.TypeOf(map[string]chan int{})
		_, err := scanType(mtyp)
		if err == nil {
			t.Error("expected error for map with unsupported value type")
		}

		mtyp2 := reflect.TypeOf(map[chan int]string{})
		_, err = scanType(mtyp2)
		if err == nil {
			t.Error("expected error for map with unsupported key type")
		}
	})

	t.Run("NamedFastPathHit", func(t *testing.T) {
		inst := newInstance()
		v := &namedMsg{msg: msg{Name: "test"}}

		// First encode (adds to cache)
		var buf bytes.Buffer
		enc := &encoder{out: &buf, tb: inst}
		if err := enc.encode(v); err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		// Second encode (hits fast path)
		if err := enc.encode(v); err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		// Use decoder too
		dec := &decoder{reader: newSliceReader(buf.Bytes()), tb: inst}
		v2 := &namedMsg{}
		if err := dec.decode(v2); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		// Second decode (hits fast path)
		dec.reader = newSliceReader(buf.Bytes())
		if err := dec.decode(v2); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
	})

	t.Run("InstanceNilErrors", func(t *testing.T) {
		e := &encoder{tb: nil}
		_, err := e.scanToCache(reflect.TypeOf(0), "")
		if err == nil {
			t.Error("expected error for nil instance in encoder")
		}

		d := &decoder{tb: nil}
		_, err = d.scanToCache(reflect.TypeOf(0), "")
		if err == nil {
			t.Error("expected error for nil instance in decoder")
		}

		// Success path
		inst := newInstance()
		e2 := &encoder{tb: inst}
		e2.scanToCache(reflect.TypeOf(0), "")
		d2 := &decoder{tb: inst}
		d2.scanToCache(reflect.TypeOf(0), "")
	})

	t.Run("NamedFastPathTypeMismatch", func(t *testing.T) {
		inst := newInstance()
		v := &namedMsg{msg: msg{Name: "test"}}

		// Register "namedMsg" for a DIFFERENT type
		type otherStruct struct{ A int }
		codec, _ := scan(reflect.TypeOf(otherStruct{}))
		inst.addSchema(reflect.TypeOf(otherStruct{}), codec, "namedMsg")

		var buf bytes.Buffer
		enc := &encoder{out: &buf, tb: inst}
		// This should NOT hit the fast path because Type() != typ
		if err := enc.encode(v); err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		dec := &decoder{reader: newSliceReader(buf.Bytes()), tb: inst}
		v2 := &namedMsg{}
		if err := dec.decode(v2); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
	})

	t.Run("MapErrorPaths", func(t *testing.T) {
		// We need a codec that fails to hit error branches in mapcodec
		fc := &FailingCodec{}
		mc := &mapcodec{keycodec: fc, valuecodec: &stringcodec{}}

		m := map[string]string{"a": "b"}
		rv := reflect.ValueOf(m)

		// encodeTo error path
		err := mc.encodeTo(newEncoder(io.Discard), rv)
		if err == nil {
			t.Error("expected error from failing codec in map encode")
		}

		// decodeTo error path (key)
		d := newDecoder(bytes.NewReader([]byte{1, 1, 'a', 1, 'b'}))
		err = mc.decodeTo(d, reflect.New(rv.Type()).Elem())
		if err == nil {
			t.Error("expected error from failing codec in map decode (key)")
		}

		// decodeTo error path (value)
		mc2 := &mapcodec{keycodec: &stringcodec{}, valuecodec: fc}
		d2 := newDecoder(bytes.NewReader([]byte{1, 1, 'a', 1, 'b'}))
		err = mc2.decodeTo(d2, reflect.New(rv.Type()).Elem())
		if err == nil {
			t.Error("expected error from failing codec in map decode (value)")
		}

		// encodeTo value error path
		err = mc2.encodeTo(newEncoder(io.Discard), rv)
		if err == nil {
			t.Error("expected error from failing codec in map encode (value)")
		}
	})

	t.Run("BinaryMarshalerErrorsExtra", func(t *testing.T) {
		mc := binaryMarshalercodec{}

		// Marshal error
		fbt := &FailingBT{}
		err := mc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(fbt))
		if err == nil {
			t.Error("expected marshal error")
		}
	})

	t.Run("FindSchemaByNameNotFound", func(t *testing.T) {
		inst := newInstance()
		_, _, found := inst.findSchemaByName("nonexistent")
		if found {
			t.Error("expected not found")
		}
	})

	t.Run("BinaryMarshalerUnaddressable", func(t *testing.T) {
		// Hit codecs.go:53: "value of type %s is not addressable"
		mc := binaryMarshalercodec{}
		v := BTCov{}             // Value version, not pointer
		rv := reflect.ValueOf(v) // Not addressable
		err := mc.encodeTo(newEncoder(io.Discard), rv)
		if err == nil {
			t.Error("expected error for unaddressable marshaler")
		}
	})

	t.Run("BinaryMarshalerManualErrors", func(t *testing.T) {
		mc := binaryMarshalercodec{}
		// Use a type that definitely doesn't implement it
		rv := reflect.ValueOf(0)
		// We need it to be addressable to reach line 57 and beyond
		ptr := reflect.New(reflect.TypeOf(0))
		rv = ptr.Elem()

		err := mc.encodeTo(newEncoder(io.Discard), rv)
		if err == nil {
			t.Error("expected error for non-implementing type in encodeTo")
		}

		err = mc.decodeTo(newDecoder(bytes.NewReader([]byte{0})), rv)
		if err == nil {
			t.Error("expected error for non-implementing type in decodeTo")
		}
	})
}

type FailingCodec struct{}

func (f *FailingCodec) encodeTo(e *encoder, rv reflect.Value) error { return io.EOF }
func (f *FailingCodec) decodeTo(d *decoder, rv reflect.Value) error { return io.EOF }
