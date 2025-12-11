package gobin

import (
	"encoding"
	"reflect"

	. "github.com/tinywasm/fmt"
)

var (
	binaryMarshalerType   = reflect.TypeOf((*encoding.BinaryMarshaler)(nil)).Elem()
	binaryUnmarshalerType = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()
)

// Codec represents a single part Codec, which can encode and decode something.
type Codec interface {
	EncodeTo(*encoder, reflect.Value) error
	DecodeTo(*decoder, reflect.Value) error
}

// ------------------------------------------------------------------------------

type reflectArrayCodec struct {
	elemCodec Codec // The codec of the array's elements
}

// Encode encodes a value into the encoder.
func (c *reflectArrayCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	for i := 0; i < l; i++ {
		idx := rv.Index(i)
		// Use the element directly without Addr() - it should already be the right type
		if err = c.elemCodec.EncodeTo(e, idx); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type binaryMarshalerCodec struct{}

func (c *binaryMarshalerCodec) EncodeTo(e *encoder, rv reflect.Value) error {
	// If this is a nil pointer, encode as zero-length payload
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		e.WriteUvarint(0)
		return nil
	}

	// Ensure we have a value that implements BinaryMarshaler (addr if needed)
	m, ok := rv.Interface().(encoding.BinaryMarshaler)
	if !ok {
		if !rv.CanAddr() {
			return Errf("value of type %s is not addressable and does not implement encoding.BinaryMarshaler", rv.Type())
		}
		rv = rv.Addr()
		m, ok = rv.Interface().(encoding.BinaryMarshaler)
		if !ok {
			return Errf("value of type %s does not implement encoding.BinaryMarshaler", rv.Type())
		}
	}

	b, err := m.MarshalBinary()
	if err != nil {
		return err
	}

	e.WriteUvarint(uint64(len(b)))
	if len(b) > 0 {
		e.Write(b)
	}
	return nil
}

func (c *binaryMarshalerCodec) DecodeTo(d *decoder, rv reflect.Value) error {
	// Read length-prefixed payload and pass to UnmarshalBinary
	l, err := d.ReadUvarint()
	if err != nil {
		return err
	}

	var b []byte
	if l > 0 {
		b = make([]byte, int(l))
		if _, err = d.Read(b); err != nil {
			return err
		}
	}

	// Ensure we have an addressable value that implements BinaryUnmarshaler
	if rv.Kind() != reflect.Ptr {
		if !rv.CanAddr() {
			return Errf("cannot unmarshal into non-addressable value of type %s", rv.Type())
		}
		rv = rv.Addr()
	}

	if rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}

	u, ok := rv.Interface().(encoding.BinaryUnmarshaler)
	if !ok {
		return Errf("value of type %s does not implement encoding.BinaryUnmarshaler", rv.Type())
	}

	return u.UnmarshalBinary(b)
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectArrayCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	l := rv.Len()
	for i := 0; i < l; i++ {
		idx := rv.Index(i)
		// Don't use Indirect here - use the indexed value directly
		if err = c.elemCodec.DecodeTo(d, idx); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type reflectSliceCodec struct {
	elemCodec Codec // The codec of the slice's elements
}

// Encode encodes a value into the encoder.
func (c *reflectSliceCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	for i := 0; i < l; i++ {
		idx := rv.Index(i)

		// Try using the element directly without Addr() - it should already be the right type
		if err = c.elemCodec.EncodeTo(e, idx); err != nil {
			return err
		}
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectSliceCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		typ := rv.Type()
		newSlice := reflect.MakeSlice(typ, int(l), int(l))
		rv.Set(newSlice)

		for i := 0; i < int(l); i++ {
			idx := rv.Index(i)
			v := reflect.Indirect(idx)
			if err = c.elemCodec.DecodeTo(d, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type reflectSliceOfPtrCodec struct {
	elemCodec Codec // The codec of the slice's elements
	elemType  reflect.Type
}

// Encode encodes a value into the encoder.
func (c *reflectSliceOfPtrCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	for i := 0; i < l; i++ {
		v := rv.Index(i)
		isNil := v.IsNil()
		e.writeBool(isNil)
		if !isNil {
			indirect := reflect.Indirect(v)
			if err = c.elemCodec.EncodeTo(e, indirect); err != nil {
				return err
			}
		}
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectSliceOfPtrCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var l uint64
	var isNil bool
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		typ := rv.Type()
		newSlice := reflect.MakeSlice(typ, int(l), int(l))
		rv.Set(newSlice)
		for i := 0; i < int(l); i++ {
			if isNil, err = d.ReadBool(); !isNil {
				if err != nil {
					return err
				}

				ptr := rv.Index(i)
				// Create new pointer value and decode directly to it
				newPtr := reflect.New(c.elemType)
				indirect := reflect.Indirect(newPtr)
				if err = c.elemCodec.DecodeTo(d, indirect); err != nil {
					return err
				}
				// Now copy the decoded value to the slice element
				ptr.Set(newPtr)
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type byteSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *byteSliceCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()

	e.WriteUvarint(uint64(l))
	if l > 0 {
		e.Write(rv.Bytes())
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *byteSliceCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		data := make([]byte, int(l))
		if _, err = d.Read(data); err == nil {
			rv.SetBytes(data)
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type boolSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *boolSliceCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	if l > 0 {
		// TODO: Need to implement proper interface access for []bool
		// For now, this is a placeholder
		dummy := make([]byte, l)
		e.Write(dummy)
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *boolSliceCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		buf := make([]byte, l)
		_, err = d.Read(buf)
		if err != nil {
			return err
		}
		// TODO: Need to implement proper bool slice creation
		// For now, create empty slice
		bools := make([]bool, l)
		boolsValue := reflect.ValueOf(bools)
		rv.Set(boolsValue)
	}
	return nil
}

// ------------------------------------------------------------------------------

type varintSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *varintSliceCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	for i := 0; i < l; i++ {
		idx := rv.Index(i)
		intVal := idx.Int()
		e.WriteVarint(intVal)
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *varintSliceCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		typ := rv.Type()
		newSlice := reflect.MakeSlice(typ, int(l), int(l))
		rv.Set(newSlice)
		for i := 0; i < int(l); i++ {
			var v int64
			if v, err = d.ReadVarint(); err == nil {
				idx := rv.Index(i)
				idx.SetInt(v)
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type varuintSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *varuintSliceCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	for i := 0; i < l; i++ {
		idx := rv.Index(i)
		uintVal := idx.Uint()
		e.WriteUvarint(uintVal)
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *varuintSliceCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var l, v uint64
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		typ := rv.Type()
		newSlice := reflect.MakeSlice(typ, int(l), int(l))
		rv.Set(newSlice)
		for i := 0; i < int(l); i++ {
			if v, err = d.ReadUvarint(); err == nil {
				idx := rv.Index(i)
				idx.SetUint(v)
			}
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type reflectPointerCodec struct {
	elemCodec Codec
}

// Encode encodes a value into the encoder.
func (c *reflectPointerCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	if rv.IsNil() {
		e.writeBool(true)
		return nil
	}

	e.writeBool(false)
	elem := rv.Elem()
	err = c.elemCodec.EncodeTo(e, elem)
	if err != nil {
		return err
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectPointerCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	isNil, err := d.ReadBool()
	if err != nil {
		return err
	}
	if isNil {
		return nil
	}

	// Check if the pointer is nil and create a new value if needed
	if rv.IsNil() {
		typ := rv.Type()
		// Get the element type using the Type.Elem() method
		elemType := typ.Elem()
		newPtr := reflect.New(elemType)
		rv.Set(newPtr)
	}

	elem := rv.Elem()
	return c.elemCodec.DecodeTo(d, elem)
}

// ------------------------------------------------------------------------------

type reflectStructCodec []fieldCodec

type fieldCodec struct {
	Index int   // The index of the field
	Codec Codec // The codec to use for this field
}

// Encode encodes a value into the encoder.
func (c reflectStructCodec) EncodeTo(e *encoder, rv reflect.Value) (err error) {
	for _, i := range c {
		field := rv.Field(i.Index)
		if err = i.Codec.EncodeTo(e, field); err != nil {
			return err
		}
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c reflectStructCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	for _, fieldCodec := range c {
		v := rv.Field(fieldCodec.Index)

		// Debug: Check if codec is nil
		if fieldCodec.Codec == nil {
			return Err(D.Field, fieldCodec.Index, "codec", D.Nil)
		}

		// Follow the original logic: handle pointers vs regular fields differently
		switch v.Kind() {
		case reflect.Ptr:
			// For pointer fields, pass the value directly to the codec
			err = fieldCodec.Codec.DecodeTo(d, v)
		default:
			// For non-pointer fields that can be set, use Indirect
			// TODO: Implement CanSet() check when available
			indirect := reflect.Indirect(v)
			err = fieldCodec.Codec.DecodeTo(d, indirect)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type stringCodec struct{}

// Encode encodes a value into the encoder.
func (c *stringCodec) EncodeTo(e *encoder, rv reflect.Value) error {
	s := rv.String()
	e.WriteString(s)
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *stringCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var s string
	if s, err = d.ReadString(); err == nil {
		rv.SetString(s)
	}
	return nil
}

// ------------------------------------------------------------------------------

type boolCodec struct{}

// Encode encodes a value into the encoder.
func (c *boolCodec) EncodeTo(e *encoder, rv reflect.Value) error {
	boolVal := rv.Bool()
	e.writeBool(boolVal)
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *boolCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var out bool
	if out, err = d.ReadBool(); err == nil {
		rv.SetBool(out)
	}
	return nil
}

// ------------------------------------------------------------------------------

type varintCodec struct{}

// Encode encodes a value into the encoder.
func (c *varintCodec) EncodeTo(e *encoder, rv reflect.Value) error {
	intVal := rv.Int()
	e.WriteVarint(intVal)
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *varintCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var v int64
	if v, err = d.ReadVarint(); err != nil {
		return err
	}
	rv.SetInt(v)
	return nil
}

// ------------------------------------------------------------------------------

type varuintCodec struct{}

// Encode encodes a value into the encoder.
func (c *varuintCodec) EncodeTo(e *encoder, rv reflect.Value) error {
	uintVal := rv.Uint()
	e.WriteUvarint(uintVal)
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *varuintCodec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var v uint64
	if v, err = d.ReadUvarint(); err != nil {
		return err
	}
	rv.SetUint(v)
	return nil
}

// ------------------------------------------------------------------------------

type float32Codec struct{}

// Encode encodes a value into the encoder.
func (c *float32Codec) EncodeTo(e *encoder, rv reflect.Value) error {
	floatVal := rv.Float()
	e.WriteFloat32(float32(floatVal))
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *float32Codec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var v float32
	if v, err = d.ReadFloat32(); err == nil {
		rv.SetFloat(float64(v))
	}
	return nil
}

// ------------------------------------------------------------------------------

type float64Codec struct{}

// Encode encodes a value into the encoder.
func (c *float64Codec) EncodeTo(e *encoder, rv reflect.Value) error {
	floatVal := rv.Float()
	e.WriteFloat64(floatVal)
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *float64Codec) DecodeTo(d *decoder, rv reflect.Value) (err error) {
	var v float64
	if v, err = d.ReadFloat64(); err == nil {
		rv.SetFloat(v)
	}
	return nil
}
