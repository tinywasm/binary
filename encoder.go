package binary

import (
	"io"
	"math"
	"reflect"

	. "github.com/tinywasm/fmt"
)

// Note: encoder pool is now managed by internal instance

// encoder represents a binary encoder.
type encoder struct {
	scratch [10]byte
	tb      *instance // Reference to the instance for schema caching
	out     io.Writer
	err     error
}

// newEncoder creates a new encoder.
func newEncoder(out io.Writer) *encoder {
	return &encoder{
		out: out,
	}
}

// reset resets the encoder and makes it ready to be reused.
func (e *encoder) reset(out io.Writer, tb *instance) {
	e.out = out
	e.err = nil
	e.tb = tb
}

// buffer returns the underlying writer.
func (e *encoder) buffer() io.Writer {
	return e.out
}

// encode encodes the value to the binary format.
func (e *encoder) encode(v any) (err error) {
	if v == nil {
		return Errf("cannot encode nil value")
	}

	// Scan the type (this will load from cache)
	rv := reflect.Indirect(reflect.ValueOf(v))
	typ := rv.Type()

	var c codec
	if c, err = e.scanToCache(typ); err == nil {
		err = c.encodeTo(e, rv)
	}

	// Double check for any error during the encode process
	if err == nil {
		err = e.err
	}
	return
}

// write writes the contents of p into the buffer.
func (e *encoder) write(p []byte) {
	if e.err == nil {
		_, e.err = e.out.Write(p)
	}
}

// writeVarint writes a variable size integer
func (e *encoder) writeVarint(v int64) {
	x := uint64(v) << 1
	if v < 0 {
		x = ^x
	}

	i := 0
	for x >= 0x80 {
		e.scratch[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	e.scratch[i] = byte(x)
	e.write(e.scratch[:(i + 1)])
}

// writeUvarint writes a variable size unsigned integer
func (e *encoder) writeUvarint(x uint64) {
	i := 0
	for x >= 0x80 {
		e.scratch[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	e.scratch[i] = byte(x)
	e.write(e.scratch[:(i + 1)])
}

// writeUint16 writes a Uint16
func (e *encoder) writeUint16(v uint16) {
	e.scratch[0] = byte(v)
	e.scratch[1] = byte(v >> 8)
	e.write(e.scratch[:2])
}

// writeUint32 writes a Uint32
func (e *encoder) writeUint32(v uint32) {
	e.scratch[0] = byte(v)
	e.scratch[1] = byte(v >> 8)
	e.scratch[2] = byte(v >> 16)
	e.scratch[3] = byte(v >> 24)
	e.write(e.scratch[:4])
}

// writeUint64 writes a Uint64
func (e *encoder) writeUint64(v uint64) {
	e.scratch[0] = byte(v)
	e.scratch[1] = byte(v >> 8)
	e.scratch[2] = byte(v >> 16)
	e.scratch[3] = byte(v >> 24)
	e.scratch[4] = byte(v >> 32)
	e.scratch[5] = byte(v >> 40)
	e.scratch[6] = byte(v >> 48)
	e.scratch[7] = byte(v >> 56)
	e.write(e.scratch[:8])
}

// writeFloat32 a 32-bit floating point number
func (e *encoder) writeFloat32(v float32) {
	e.writeUint32(math.Float32bits(v))
}

// writeFloat64 a 64-bit floating point number
func (e *encoder) writeFloat64(v float64) {
	e.writeUint64(math.Float64bits(v))
}

// writeBool writes a single boolean value into the buffer
func (e *encoder) writeBool(v bool) {
	e.scratch[0] = 0
	if v {
		e.scratch[0] = 1
	}
	e.write(e.scratch[:1])
}

// writeString writes a string prefixed with a variable-size integer size.
func (e *encoder) writeString(v string) {
	e.writeUvarint(uint64(len(v)))
	e.write(toBytes(v))
}

// scanToCache scans the type and caches it in the internal instance
func (e *encoder) scanToCache(t reflect.Type) (codec, error) {
	if e.tb == nil {
		return nil, Err("encoder", "scanToCache", "instance", "nil")
	}

	// Use the instance's schema caching mechanism
	return e.tb.scanToCache(t)
}
