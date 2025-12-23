package binary

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestCoverageGaps(t *testing.T) {
	// SetLog
	SetLog(func(msg ...any) {})
	SetLog(nil)

	// Encode/Decode errors and branches
	t.Run("EncodeDecodeBranches", func(t *testing.T) {
		var b []byte
		// Encode to invalid output
		err := Encode(1, 1)
		if err == nil {
			t.Error("Expected error encoding to int")
		}

		// Decode from invalid input
		err = Decode(1, &b)
		if err == nil {
			t.Error("Expected error decoding from int")
		}

		// Decode from io.Reader
		var out string
		err = Decode(bytes.NewReader([]byte{4, 't', 'e', 's', 't'}), &out)
		if err != nil {
			t.Errorf("Unexpected error decoding from reader: %v", err)
		}
		if out != "test" {
			t.Errorf("Expected 'test', got %v", out)
		}

		// Encode to io.Writer
		var buf bytes.Buffer
		err = Encode("test", &buf)
		if err != nil {
			t.Errorf("Unexpected error encoding to writer: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), []byte{4, 't', 'e', 's', 't'}) {
			t.Errorf("Expected encoded 'test', got %v", buf.Bytes())
		}
	})

	t.Run("CodecCoverage", func(t *testing.T) {
		// boolSliceCodec
		type BoolSlice struct {
			B []bool
		}
		bs := &BoolSlice{B: []bool{true, false, true}}
		var data []byte
		err := Encode(bs, &data)
		if err != nil {
			t.Errorf("Encode boolSlice failed: %v", err)
		}
		var bs2 BoolSlice
		err = Decode(data, &bs2)
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

	t.Run("EncoderDecoderInternals", func(t *testing.T) {
		e := newEncoder(io.Discard)
		if e.buffer() != io.Discard {
			t.Error("buffer() returned wrong writer")
		}

		// writeUint16
		var buf bytes.Buffer
		e = newEncoder(&buf)
		e.writeUint16(0x1234)
		if !bytes.Equal(buf.Bytes(), []byte{0x34, 0x12}) {
			t.Errorf("writeUint16 failed, got %v", buf.Bytes())
		}

		// readUint16
		d := newDecoder(bytes.NewReader([]byte{0x34, 0x12}))
		val, err := d.readUint16()
		if err != nil || val != 0x1234 {
			t.Errorf("readUint16 failed: %v, %v", val, err)
		}

		// scanToCache errors
		_, err = e.scanToCache(nil)
		if err == nil {
			t.Error("Expected error scanning nil type")
		}

		d = newDecoder(nil)
		_, err = d.scanToCache(reflect.TypeOf(0))
		if err == nil {
			t.Error("Expected error scanning with nil tb")
		}
	})

	t.Run("ScannerUnsupportedTypes", func(t *testing.T) {
		types := []reflect.Type{
			reflect.TypeOf(make(chan int)),
			reflect.TypeOf(func() {}),
		}
		for _, typ := range types {
			_, err := scanType(typ)
			if err == nil {
				t.Errorf("Expected error for unsupported type %v", typ)
			}
		}
	})

	t.Run("SchemaEviction", func(t *testing.T) {
		inst := newInstance()
		// Loop more to ensure eviction
		for i := 1; i <= 1005; i++ {
			typ := reflect.ArrayOf(i, reflect.TypeOf(byte(0)))
			inst.scanToCache(typ)
		}
		if len(inst.schemas) > 1000 {
			t.Errorf("Cache eviction failed, length: %d", len(inst.schemas))
		}
	})

	t.Run("ScannerEdgeCases", func(t *testing.T) {
		_, err := scanType(nil)
		if err == nil {
			t.Error("Expected error for nil type in scanType")
		}

		d := newDecoder(bytes.NewReader(nil))
		c := byteSlicecodec{}
		err = c.decodeTo(d, reflect.ValueOf(1))
		if err == nil {
			t.Error("Expected error for non-slice in byteSlicecodec")
		}

		bc := boolSlicecodec{}
		err = bc.decodeTo(d, reflect.ValueOf(1))
		if err == nil {
			t.Error("Expected error for non-slice in boolSlicecodec")
		}
	})

	t.Run("BinaryInternalsExtra", func(t *testing.T) {
		inst := newInstance(func(msg ...any) {})
		if inst.log == nil {
			t.Error("expected log function to be set")
		}

		err := inst.encodeTo(nil, io.Discard)
		if err == nil {
			t.Error("expected error encoding nil data")
		}

		var result string
		d := inst.decoders.Get().(*decoder)
		d.reader = newStreamReader(bytes.NewReader(nil))
		inst.decoders.Put(d)
		inst.decodeFrom(bytes.NewReader([]byte{4, 'a', 'b', 'c', 'd'}), &result)
		if result != "abcd" {
			t.Errorf("expected abcd, got %v", result)
		}

		d = inst.decoders.Get().(*decoder)
		d.reader = newSliceReader(nil)
		inst.decoders.Put(d)
		inst.decodeFrom(bytes.NewReader([]byte{4, 'a', 'b', 'c', 'd'}), &result)

		err = inst.decodeFrom(bytes.NewReader(nil), &result)
		if err == nil {
			t.Error("expected error decoding from empty reader")
		}

		var out []byte
		err = Encode(nil, &out)
		if err == nil {
			t.Error("expected error encoding nil into byte slice")
		}
	})

	t.Run("ReaderExtra", func(t *testing.T) {
		newReader(nil)
		newReader(&bytes.Buffer{})
		r := newSliceReader(nil)
		newReader(r)

		r2 := newSliceReader([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02})
		_, err := r2.ReadUvarint()
		if err == nil {
			t.Error("expected overflow error")
		}
	})

	t.Run("CodecsExtra", func(t *testing.T) {
		ac := reflectArraycodec{elemcodec: &stringcodec{}}
		eErr := newEncoder(io.Discard)
		eErr.err = io.EOF
		ac.encodeTo(eErr, reflect.ValueOf([1]string{"a"}))
		if eErr.err != io.EOF {
			t.Errorf("expected io.EOF, got %v", eErr.err)
		}

		mc := binaryMarshalercodec{}
		var bmc *BTCov
		err := mc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(bmc))
		if err != nil {
			t.Error(err)
		}

		pc := reflectPointercodec{elemcodec: &stringcodec{}}
		var s *string
		err = pc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(s))
		if err != nil {
			t.Error(err)
		}

		spc := reflectSliceOfPtrcodec{elemcodec: &stringcodec{}, elemType: reflect.TypeOf("")}
		var ss []*string
		err = spc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(ss))
		if err != nil {
			t.Error(err)
		}

		d := newDecoder(bytes.NewReader([]byte{0}))
		err = spc.decodeTo(d, reflect.ValueOf(&ss).Elem())
		if err != nil {
			t.Error(err)
		}

		sc := reflectStructcodec{
			{Index: 0, codec: &stringcodec{}},
		}
		eS := newEncoder(io.Discard)
		eS.err = io.EOF
		sc.encodeTo(eS, reflect.ValueOf(struct{ S string }{"a"}))
		if eS.err != io.EOF {
			t.Errorf("expected io.EOF, got %v", eS.err)
		}

		dErr := newDecoder(bytes.NewReader(nil))
		err = sc.decodeTo(dErr, reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in reflectStructcodec due to type mismatch")
		}
	})

	t.Run("MarshallerCoverage", func(t *testing.T) {
		type MarshalerStruct struct {
			BT BTCov
		}
		ms := &MarshalerStruct{}
		var data []byte
		Encode(ms, &data)
		Decode(data, ms)

		_, err := scanType(reflect.TypeOf(BTCov{}))
		if err != nil {
			t.Errorf("expected no error for BTCov, got %v", err)
		}
	})

	t.Run("ScanTypeErrors", func(t *testing.T) {
		type PtrErr *chan int
		_, err := scanType(reflect.TypeOf(PtrErr(nil)))
		if err == nil {
			t.Error("expected error for ptr to chan")
		}

		type ArrayErr [1]chan int
		_, err = scanType(reflect.TypeOf(ArrayErr{}))
		if err == nil {
			t.Error("expected error for array of chan")
		}

		type SliceErr []chan int
		_, err = scanType(reflect.TypeOf(SliceErr{}))
		if err == nil {
			t.Error("expected error for slice of chan")
		}

		type SlicePtrErr []*chan int
		_, err = scanType(reflect.TypeOf(SlicePtrErr{}))
		if err == nil {
			t.Error("expected error for slice of ptr to chan")
		}

		type StructErr struct {
			C chan int
		}
		_, err = scanType(reflect.TypeOf(StructErr{}))
		if err == nil {
			t.Error("expected error for struct with chan")
		}

		inst := newInstance()
		typ := reflect.TypeOf(0)
		inst.scanToCache(typ)
		inst.scanToCache(typ) // Double scan to hit findSchema

		_, err = inst.scanToCache(reflect.TypeOf(make(chan int)))
		if err == nil {
			t.Error("expected error scanning chan")
		}

	})

	t.Run("CodecErrors", func(t *testing.T) {
		dErr := newDecoder(bytes.NewReader(nil)) // EOF

		// reflectArraycodec error
		ac := reflectArraycodec{elemcodec: &stringcodec{}}
		err := ac.decodeTo(dErr, reflect.ValueOf([1]string{}))
		if err == nil {
			t.Error("expected error in reflectArraycodec.decodeTo")
		}

		// binaryMarshalercodec decode error with nil ptr
		mc := binaryMarshalercodec{}
		var bmc *BTCov
		// Need a pointer to the pointer to be able to Set it
		rv := reflect.ValueOf(&bmc).Elem()
		// We need data to pass readSlice
		dData := newDecoder(bytes.NewReader([]byte{4, 1, 2, 3, 4}))
		err = mc.decodeTo(dData, rv)
		if err != nil {
			t.Errorf("unexpected error in binaryMarshalercodec.decodeTo: %v", err)
		}
		if bmc == nil {
			t.Error("expected bmc to be initialized")
		}

		// Non-pointer marshaller case (if it implement on value receiver, but here it is on pointer)
		// Let's try to decode into a value BTCov
		var bmcV BTCov
		rvV := reflect.ValueOf(&bmcV).Elem()
		dDataV := newDecoder(bytes.NewReader([]byte{4, 1, 2, 3, 4}))
		err = mc.decodeTo(dDataV, rvV)
		if err != nil {
			t.Errorf("unexpected error in binaryMarshalercodec.decodeTo (value): %v", err)
		}

		// reflectSliceOfPtrcodec decode error
		spc := reflectSliceOfPtrcodec{elemcodec: &stringcodec{}, elemType: reflect.TypeOf("")}
		var ss []*string
		err = spc.decodeTo(dErr, reflect.ValueOf(&ss).Elem())
		if err == nil {
			t.Error("expected error in reflectSliceOfPtrcodec.decodeTo")
		}

		// Kind checks encode
		err = ac.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in reflectArraycodec.encodeTo")
		}

		pcc := reflectPointercodec{}
		err = pcc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in reflectPointercodec.encodeTo")
		}

		slc := reflectSlicecodec{}
		err = slc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in reflectSlicecodec.encodeTo")
		}

		spcc := reflectSliceOfPtrcodec{}
		err = spcc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in reflectSliceOfPtrcodec.encodeTo")
		}
	})
}

type BTCov struct{}

func (b *BTCov) MarshalBinary() ([]byte, error) { return nil, nil }
func (b *BTCov) UnmarshalBinary([]byte) error   { return nil }
