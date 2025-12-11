package gobin

import (
	"io"
	"math"
	"reflect"

	. "github.com/tinywasm/fmt"
)

// Note: decoder pool is now managed by GoBin instance

// decoder represents a binary decoder.
type decoder struct {
	reader reader
	tb     *GoBin // Reference to the GoBin instance for schema caching
}

// NewDecoder creates a binary decoder (deprecated - use GoBin instance methods).
func NewDecoder(r io.Reader) *decoder {
	return &decoder{
		reader: newReader(r),
	}
}

// Decode decodes a value by reading from the underlying io.Reader.
func (d *decoder) Decode(v any) (err error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	canAddr := rv.CanAddr()
	if !canAddr {
		return Err(D.Binary, "decoder", D.Required, D.Type, D.Pointer)
	}

	// Scan the type (this will load from cache)
	var c Codec
	if c, err = d.scanToCache(rv.Type()); err == nil {
		err = c.DecodeTo(d, rv)
	}

	return
}

// Read reads a set of bytes
func (d *decoder) Read(b []byte) (int, error) {
	return d.reader.Read(b)
}

// ReadUvarint reads a variable-length Uint64 from the buffer.
func (d *decoder) ReadUvarint() (uint64, error) {
	return d.reader.ReadUvarint()
}

// ReadVarint reads a variable-length Int64 from the buffer.
func (d *decoder) ReadVarint() (int64, error) {
	return d.reader.ReadVarint()
}

// ReadUint16 reads a uint16
func (d *decoder) ReadUint16() (out uint16, err error) {
	var b []byte
	if b, err = d.reader.Slice(2); err == nil {
		_ = b[1] // bounds check hint to compiler
		out = (uint16(b[0]) | uint16(b[1])<<8)
	}
	return
}

// ReadUint32 reads a uint32
func (d *decoder) ReadUint32() (out uint32, err error) {
	var b []byte
	if b, err = d.reader.Slice(4); err == nil {
		_ = b[3] // bounds check hint to compiler
		out = (uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
	}
	return
}

// ReadUint64 reads a uint64
func (d *decoder) ReadUint64() (out uint64, err error) {
	var b []byte
	if b, err = d.reader.Slice(8); err == nil {
		_ = b[7] // bounds check hint to compiler
		out = (uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
	}
	return
}

// ReadFloat32 reads a float32
func (d *decoder) ReadFloat32() (out float32, err error) {
	var v uint32
	if v, err = d.ReadUint32(); err == nil {
		out = math.Float32frombits(v)
	}
	return
}

// ReadFloat64 reads a float64
func (d *decoder) ReadFloat64() (out float64, err error) {
	var v uint64
	if v, err = d.ReadUint64(); err == nil {
		out = math.Float64frombits(v)
	}
	return
}

// ReadBool reads a single boolean value from the slice.
func (d *decoder) ReadBool() (bool, error) {
	b, err := d.reader.ReadByte()
	return b == 1, err
}

// ReadString a string prefixed with a variable-size integer size.
func (d *decoder) ReadString() (out string, err error) {
	var b []byte
	if b, err = d.ReadSlice(); err == nil {
		out = string(b)
	}
	return
}

// Slice selects a sub-slice of next bytes. This is similar to Read() but does not
// actually perform a copy, but simply uses the underlying slice (if available) and
// returns a sub-slice pointing to the same array. Since this requires access
// to the underlying data, this is only available for a slice reader.
func (d *decoder) Slice(n int) ([]byte, error) {
	return d.reader.Slice(n)
}

// ReadSlice reads a varint prefixed sub-slice without copying and returns the underlying
// byte slice.
func (d *decoder) ReadSlice() (b []byte, err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		b, err = d.Slice(int(l))
	}
	return
}

// Reset resets the decoder and makes it ready to be reused.
func (d *decoder) Reset(data []byte, tb *GoBin) {
	if d.reader == nil {
		d.reader = newSliceReader(data)
	} else {
		d.reader.(*sliceReader).Reset(data)
	}
	d.tb = tb
}

// scanToCache scans the type and caches it in the GoBin instance
func (d *decoder) scanToCache(t reflect.Type) (Codec, error) {
	if d.tb == nil {
		return nil, Err("decoder", "scanToCache", "GoBin", "nil")
	}

	// Use the GoBin instance's schema caching mechanism
	return d.tb.scanToCache(t)
}
