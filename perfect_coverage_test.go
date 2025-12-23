package binary

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

// EvilWriter always fails
type EvilWriter struct{}

func (e *EvilWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("evil write error")
}

// EvilReader always fails
type EvilReader struct{}

func (e *EvilReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("evil read error")
}

// FailingBT implements Marshaler/Unmarshaler but fails
type FailingBT struct{}

func (f *FailingBT) MarshalBinary() ([]byte, error) {
	return nil, errors.New("marshal error")
}
func (f *FailingBT) UnmarshalBinary(data []byte) error {
	return errors.New("unmarshal error")
}

func TestPerfectCoverage(t *testing.T) {
	t.Run("FailingCodecs", func(t *testing.T) {
		// reflectArraycodec error in elements
		ac := reflectArraycodec{elemcodec: &stringcodec{}}
		eErr := newEncoder(&EvilWriter{})
		err := ac.encodeTo(eErr, reflect.ValueOf([1]string{"a"}))
		if err == nil {
			t.Error("expected evil write error in reflectArraycodec")
		}

		// binaryMarshalercodec failures
		mc := binaryMarshalercodec{}
		fbt := &FailingBT{}
		// Test nil pointer case
		var nilFbt *FailingBT
		err = mc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(nilFbt))
		if err != nil {
			t.Error(err)
		}

		err = mc.encodeTo(newEncoder(&EvilWriter{}), reflect.ValueOf(fbt))
		if err == nil {
			t.Error("expected marshal error")
		}

		dF := newDecoder(bytes.NewReader([]byte{1, 1})) // data exists
		err = mc.decodeTo(dF, reflect.ValueOf(fbt))
		if err == nil {
			t.Error("expected unmarshal error")
		}

		// decodeTo with readSlice error
		err = mc.decodeTo(newDecoder(&EvilReader{}), reflect.ValueOf(fbt))
		if err == nil {
			t.Error("expected read error in binaryMarshalercodec")
		}

		// Marshaller decode length-prefixed payload read error
		dFailLarge := newDecoder(bytes.NewReader([]byte{100})) // length 100
		err = mc.decodeTo(dFailLarge, reflect.ValueOf(fbt))
		if err == nil {
			t.Error("expected read error for payload")
		}

		// Marshaller non-addressable and non-implementing
		err = mc.decodeTo(newDecoder(bytes.NewReader([]byte{0})), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error for non-addressable int")
		}

		var i int
		err = mc.decodeTo(newDecoder(bytes.NewReader([]byte{0})), reflect.ValueOf(&i).Elem())
		if err == nil {
			t.Error("expected error for addressable int not implementing Unmarshaler")
		}

		// encodeTo non-addressable and non-implementing
		err = mc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error")
		}
		err = mc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(&i).Elem())
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("FailingStructAndSlice", func(t *testing.T) {
		// reflectStructcodec Kind check
		sc := reflectStructcodec{{Index: 0, codec: &stringcodec{}}}
		err := sc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in reflectStructcodec Kind check")
		}

		// reflectStructcodec element error
		type S struct{ Name string }
		s := S{Name: "test"}
		err = sc.encodeTo(newEncoder(&EvilWriter{}), reflect.ValueOf(s))
		if err == nil {
			t.Error("expected evil write error in reflectStructcodec")
		}

		// reflectStructcodec nil codec
		scNil := reflectStructcodec{{Index: 0, codec: nil}}
		err = scNil.decodeTo(newDecoder(bytes.NewReader(nil)), reflect.ValueOf(&s).Elem())
		if err == nil {
			t.Error("expected error for nil codec in struct")
		}

		// reflectSlicecodec error
		slc := reflectSlicecodec{elemcodec: &stringcodec{}}
		err = slc.encodeTo(newEncoder(&EvilWriter{}), reflect.ValueOf([]string{"a"}))
		if err == nil {
			t.Error("expected evil write error in reflectSlicecodec")
		}

		// reflectSliceOfPtrcodec error
		spc := reflectSliceOfPtrcodec{elemcodec: &stringcodec{}, elemType: reflect.TypeOf("")}
		v := "a"
		err = spc.encodeTo(newEncoder(&EvilWriter{}), reflect.ValueOf([]*string{&v}))
		if err == nil {
			t.Error("expected evil write error in reflectSliceOfPtrcodec")
		}

		// decode error branches
		err = slc.decodeTo(newDecoder(&EvilReader{}), reflect.ValueOf(&[]string{}).Elem())
		if err == nil {
			t.Error("expected read error in reflectSlicecodec")
		}

		// Element decode error in slice
		err = slc.decodeTo(newDecoder(bytes.NewReader([]byte{1})), reflect.ValueOf(&[]string{}).Elem())
		if err == nil {
			t.Error("expected element decode error")
		}

		err = spc.decodeTo(newDecoder(&EvilReader{}), reflect.ValueOf(&[]*string{}).Elem())
		if err == nil {
			t.Error("expected read error in reflectSliceOfPtrcodec")
		}

		// ByteSliceCodec decode error
		bsc := byteSlicecodec{}
		err = bsc.decodeTo(newDecoder(&EvilReader{}), reflect.ValueOf(&[]byte{}).Elem())
		if err == nil {
			t.Error("expected read error in byteSlicecodec")
		}

		// boolSlicecodec decode error
		blc := boolSlicecodec{}
		err = blc.decodeTo(newDecoder(&EvilReader{}), reflect.ValueOf(&[]bool{}).Elem())
		if err == nil {
			t.Error("expected read error in boolSlicecodec")
		}

		// Inner loop error for boolSlicecodec
		err = blc.decodeTo(newDecoder(bytes.NewReader([]byte{1})), reflect.ValueOf(&[]bool{}).Elem())
		if err == nil {
			t.Error("expected inner loop error")
		}

		// Empty slices coverage
		emptyData := []byte{0} // length 0
		err = slc.decodeTo(newDecoder(bytes.NewReader(emptyData)), reflect.ValueOf(&[]string{}).Elem())
		if err != nil {
			t.Error(err)
		}
		err = spc.decodeTo(newDecoder(bytes.NewReader(emptyData)), reflect.ValueOf(&[]*string{}).Elem())
		if err != nil {
			t.Error(err)
		}
		err = bsc.decodeTo(newDecoder(bytes.NewReader(emptyData)), reflect.ValueOf(&[]byte{}).Elem())
		if err != nil {
			t.Error(err)
		}
		err = blc.decodeTo(newDecoder(bytes.NewReader(emptyData)), reflect.ValueOf(&[]bool{}).Elem())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("NumericErrors", func(t *testing.T) {
		dE := newDecoder(&EvilReader{})

		var b bool
		if err := (new(boolcodec)).decodeTo(dE, reflect.ValueOf(&b).Elem()); err == nil {
			t.Error("expected error")
		}
		var i int64
		if err := (new(varintcodec)).decodeTo(dE, reflect.ValueOf(&i).Elem()); err == nil {
			t.Error("expected error")
		}
		var u uint64
		if err := (new(varuintcodec)).decodeTo(dE, reflect.ValueOf(&u).Elem()); err == nil {
			t.Error("expected error")
		}
		var f32 float32
		if err := (new(float32codec)).decodeTo(dE, reflect.ValueOf(&f32).Elem()); err == nil {
			t.Error("expected error")
		}
		var f64 float64
		if err := (new(float64codec)).decodeTo(dE, reflect.ValueOf(&f64).Elem()); err == nil {
			t.Error("expected error")
		}

		// slice errors for coverage
		if err := (new(varintSlicecodec)).decodeTo(dE, reflect.ValueOf(&[]int64{}).Elem()); err == nil {
			t.Error("expected error")
		}
		// Inner loop error for varintSlicecodec
		if err := (new(varintSlicecodec)).decodeTo(newDecoder(bytes.NewReader([]byte{1})), reflect.ValueOf(&[]int64{}).Elem()); err == nil {
			t.Error("expected inner loop error")
		}

		if err := (new(varuintSlicecodec)).decodeTo(dE, reflect.ValueOf(&[]uint64{}).Elem()); err == nil {
			t.Error("expected error")
		}
		// Inner loop error for varuintSlicecodec
		if err := (new(varuintSlicecodec)).decodeTo(newDecoder(bytes.NewReader([]byte{1})), reflect.ValueOf(&[]uint64{}).Elem()); err == nil {
			t.Error("expected inner loop error")
		}

		// Empty slices
		dEmpty := newDecoder(bytes.NewReader([]byte{0}))
		if err := (new(varintSlicecodec)).decodeTo(dEmpty, reflect.ValueOf(&[]int64{}).Elem()); err != nil {
			t.Error(err)
		}
		dEmpty2 := newDecoder(bytes.NewReader([]byte{0}))
		if err := (new(varuintSlicecodec)).decodeTo(dEmpty2, reflect.ValueOf(&[]uint64{}).Elem()); err != nil {
			t.Error(err)
		}
	})

	t.Run("BinaryGaps", func(t *testing.T) {
		inst := newInstance()
		// decodeFrom
		var res string
		inst.decodeFrom(bytes.NewReader([]byte{4, 'a', 'b', 'c', 'd'}), &res)

		// scanToCache(nil) directly
		_, err := inst.scanToCache(nil)
		if err == nil {
			t.Error("expected error scanning nil type")
		}

		// findSchema hit
		typ := reflect.TypeOf(0)
		inst.scanToCache(typ)
		inst.scanToCache(typ)

		// scanType error
		_, err = inst.scanToCache(reflect.TypeOf(make(chan int)))
		if err == nil {
			t.Error("expected scanType error")
		}

		// Kind checks for slices
		dErr := newDecoder(bytes.NewReader(nil))
		vuc := varuintSlicecodec{}
		err = vuc.decodeTo(dErr, reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in varuintSlicecodec Kind check")
		}

		vic := varintSlicecodec{}
		err = vic.decodeTo(dErr, reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in varintSlicecodec Kind check")
		}

		err = vuc.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in varuintSlicecodec Kind check")
		}

		err = vic.encodeTo(newEncoder(io.Discard), reflect.ValueOf(1))
		if err == nil {
			t.Error("expected error in varintSlicecodec Kind check")
		}
	})

	t.Run("StructPtrCoverage", func(t *testing.T) {
		type P struct {
			S *string
		}
		sc := reflectStructcodec{{Index: 0, codec: &reflectPointercodec{elemcodec: &stringcodec{}}}}
		var p P
		d := newDecoder(bytes.NewReader([]byte{0, 4, 't', 'e', 's', 't'})) // isNil=false, string="test"
		err := sc.decodeTo(d, reflect.ValueOf(&p).Elem())
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if p.S == nil || *p.S != "test" {
			t.Errorf("Expected 'test', got %v", p.S)
		}

		// Struct decode error branch (codec returns error)
		dFail := newDecoder(&EvilReader{})
		err = sc.decodeTo(dFail, reflect.ValueOf(&p).Elem())
		if err == nil {
			t.Error("expected error in struct member decode")
		}

		// reflectSliceOfPtrcodec failing readBool
		spc := reflectSliceOfPtrcodec{elemcodec: &stringcodec{}, elemType: reflect.TypeOf("")}
		var ss []*string
		dFail2 := newDecoder(bytes.NewReader([]byte{1})) // length 1, then EOF for bool
		err = spc.decodeTo(dFail2, reflect.ValueOf(&ss).Elem())
		if err == nil {
			t.Error("expected error in reflectSliceOfPtrcodec readBool")
		}

		// reflectSliceOfPtrcodec element decode error
		dFail3 := newDecoder(bytes.NewReader([]byte{1, 0})) // length 1, isNil=false, then EOF for element
		err = spc.decodeTo(dFail3, reflect.ValueOf(&ss).Elem())
		if err == nil {
			t.Error("expected error in reflectSliceOfPtrcodec element decode")
		}
	})
}
