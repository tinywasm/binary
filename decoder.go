package binary

import (
	"io"
	"math"
	"reflect"

	. "github.com/tinywasm/fmt"
)

// Note: decoder pool is now managed by internal instance

// decoder represents a binary decoder.
type decoder struct {
	scratch [10]byte
	reader  reader
	tb      *instance // Reference to the instance for schema caching
}

// newDecoder creates a binary decoder.
func newDecoder(r io.Reader) *decoder {
	return &decoder{
		reader: newReader(r),
	}
}

// decode decodes a value by reading from the underlying io.Reader.
func (d *decoder) decode(v any) (err error) {
	// Fast path: check if type implements namedHandler
	if nh, ok := v.(namedHandler); ok {
		name := nh.HandlerName()
		if c, typ, found := d.tb.findSchemaByName(name); found {
			rv := reflect.Indirect(reflect.ValueOf(v))
			if rv.Type() == typ {
				return c.decodeTo(d, rv)
			}
		}
	}

	rv := reflect.Indirect(reflect.ValueOf(v))
	canAddr := rv.CanAddr()
	if !canAddr {
		return Err(D.Binary, "decoder", D.Required, D.Type, D.Pointer)
	}

	// Scan the type (this will load from cache)
	var c codec
	typ := rv.Type()
	var name string
	if nh, ok := v.(namedHandler); ok {
		name = nh.HandlerName()
	}

	if c, err = d.scanToCache(typ, name); err == nil {
		err = c.decodeTo(d, rv)
	}

	return
}

// read reads a set of bytes
func (d *decoder) read(b []byte) (int, error) {
	return d.reader.Read(b)
}

// readUvarint reads a variable-length Uint64 from the buffer.
func (d *decoder) readUvarint() (uint64, error) {
	return d.reader.ReadUvarint()
}

// readVarint reads a variable-length Int64 from the buffer.
func (d *decoder) readVarint() (int64, error) {
	return d.reader.ReadVarint()
}

// readUint16 reads a uint16
func (d *decoder) readUint16() (out uint16, err error) {
	var b []byte
	if b, err = d.reader.Slice(2); err == nil {
		_ = b[1] // bounds check hint to compiler
		out = (uint16(b[0]) | uint16(b[1])<<8)
	}
	return
}

// readUint32 reads a uint32
func (d *decoder) readUint32() (out uint32, err error) {
	var b []byte
	if b, err = d.reader.Slice(4); err == nil {
		_ = b[3] // bounds check hint to compiler
		out = (uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
	}
	return
}

// readUint64 reads a uint64
func (d *decoder) readUint64() (out uint64, err error) {
	var b []byte
	if b, err = d.reader.Slice(8); err == nil {
		_ = b[7] // bounds check hint to compiler
		out = (uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
	}
	return
}

// readFloat32 reads a float32
func (d *decoder) readFloat32() (out float32, err error) {
	var v uint32
	if v, err = d.readUint32(); err == nil {
		out = math.Float32frombits(v)
	}
	return
}

// readFloat64 reads a float64
func (d *decoder) readFloat64() (out float64, err error) {
	var v uint64
	if v, err = d.readUint64(); err == nil {
		out = math.Float64frombits(v)
	}
	return
}

// readBool reads a single boolean value from the slice.
func (d *decoder) readBool() (bool, error) {
	b, err := d.reader.ReadByte()
	return b == 1, err
}

// readString a string prefixed with a variable-size integer size.
func (d *decoder) readString() (out string, err error) {
	var l uint64
	if l, err = d.readUvarint(); err == nil && l > 0 {
		var b []byte
		if l <= 10 {
			b = d.scratch[:l]
			if _, err = d.read(b); err == nil {
				out = string(b)
			}
		} else {
			if b, err = d.slice(int(l)); err == nil {
				out = string(b)
			}
		}
	}
	return
}

// slice selects a sub-slice of next bytes. This is similar to Read() but does not
// actually perform a copy, but simply uses the underlying slice (if available) and
// returns a sub-slice pointing to the same array. Since this requires access
// to the underlying data, this is only available for a slice reader.
func (d *decoder) slice(n int) ([]byte, error) {
	return d.reader.Slice(n)
}

// readSlice reads a varint prefixed sub-slice without copying and returns the underlying
// byte slice.
func (d *decoder) readSlice() (b []byte, err error) {
	var l uint64
	if l, err = d.readUvarint(); err == nil {
		b, err = d.slice(int(l))
	}
	return
}

// reset resets the decoder and makes it ready to be reused.
func (d *decoder) reset(data []byte, tb *instance) {
	if d.reader == nil {
		d.reader = newSliceReader(data)
	} else {
		if sr, ok := d.reader.(*sliceReader); ok {
			sr.Reset(data)
		} else {
			d.reader = newSliceReader(data)
		}
	}
	d.tb = tb
}

// scanToCache scans the type and caches it in the internal instance
func (d *decoder) scanToCache(t reflect.Type, name string) (codec, error) {
	if d.tb == nil {
		return nil, Err("decoder", "scanToCache", "instance", "nil")
	}

	// Use the instance's schema caching mechanism
	return d.tb.scanToCache(t, name)
}
