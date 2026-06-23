package binary

import (
	"io"
	"math"

	"github.com/tinywasm/fmt"
)

type binaryWriter struct {
	out     io.Writer
	scratch [10]byte
	err     error
}

func (w *binaryWriter) reset(out io.Writer) {
	w.out = out
	w.err = nil
}

func newWriter(out io.Writer) *binaryWriter {
	w := &binaryWriter{out: out}
	return w
}

// FieldWriter implementation

func (w *binaryWriter) String(name, val string) {
	w.writeUvarint(uint64(len(val)))
	w.write([]byte(val))
}

func (w *binaryWriter) Raw(name, val string) {
	w.String(name, val)
}

func (w *binaryWriter) Int(name string, val int64) {
	w.writeVarint(val)
}

func (w *binaryWriter) Uint(name string, val uint64) {
	w.writeUvarint(val)
}

func (w *binaryWriter) Float(name string, val float64) {
	bits := math.Float64bits(val)
	w.scratch[0] = byte(bits)
	w.scratch[1] = byte(bits >> 8)
	w.scratch[2] = byte(bits >> 16)
	w.scratch[3] = byte(bits >> 24)
	w.scratch[4] = byte(bits >> 32)
	w.scratch[5] = byte(bits >> 40)
	w.scratch[6] = byte(bits >> 48)
	w.scratch[7] = byte(bits >> 56)
	w.write(w.scratch[:8])
}

func (w *binaryWriter) Bool(name string, val bool) {
	w.scratch[0] = 0
	if val {
		w.scratch[0] = 1
	}
	w.write(w.scratch[:1])
}

func (w *binaryWriter) Bytes(name string, val []byte) {
	w.writeUvarint(uint64(len(val)))
	w.write(val)
}

func (w *binaryWriter) Null(name string) {
	w.scratch[0] = 0
	w.write(w.scratch[:1])
}

func (w *binaryWriter) Object(name string, val fmt.Encodable) {
	if val != nil && !val.IsNil() {
		w.scratch[0] = 1
		w.write(w.scratch[:1])
		val.EncodeFields(w)
	} else {
		w.Null(name)
	}
}

func (w *binaryWriter) Array(name string, n int) fmt.ArrayWriter {
	w.writeUvarint(uint64(n))
	return &binaryArrayWriter{w: w}
}

// ArrayWriter implementation

type binaryArrayWriter struct {
	w *binaryWriter
}

func (w *binaryArrayWriter) String(val string) {
	w.w.String("", val)
}

func (w *binaryArrayWriter) Int(val int64) {
	w.w.Int("", val)
}

func (w *binaryArrayWriter) Float(val float64) {
	w.w.Float("", val)
}

func (w *binaryArrayWriter) Bool(val bool) {
	w.w.Bool("", val)
}

func (w *binaryArrayWriter) Bytes(val []byte) {
	w.w.Bytes("", val)
}

func (w *binaryArrayWriter) Object(val fmt.Encodable) {
	w.w.Object("", val)
}

func (w *binaryArrayWriter) Close() {
	// binary protocol does not need a closing delimiter
}

// Internal helpers

func (w *binaryWriter) write(p []byte) {
	if w.err == nil {
		_, w.err = w.out.Write(p)
	}
}

func (w *binaryWriter) writeVarint(v int64) {
	x := uint64(v) << 1
	if v < 0 {
		x = ^x
	}
	w.writeUvarint(x)
}

func (w *binaryWriter) writeUvarint(x uint64) {
	i := 0
	for x >= 0x80 {
		w.scratch[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	w.scratch[i] = byte(x)
	w.write(w.scratch[:(i + 1)])
}

// --- Reader ---

type binaryReader struct {
	r reader
}

func (br *binaryReader) reset(r io.Reader) {
	br.r = newReader(r)
}

func newBinaryReader(r io.Reader) *binaryReader {
	br := &binaryReader{r: newReader(r)}
	return br
}

// FieldReader implementation

func (br *binaryReader) String(name string) (string, bool) {
	l, err := br.r.ReadUvarint()
	if err != nil {
		return "", false
	}
	if l == 0 {
		return "", true
	}
	b, err := br.r.Slice(int(l))
	if err != nil {
		return "", false
	}
	return string(b), true
}

func (br *binaryReader) Raw(name string) (string, bool) {
	return br.String(name)
}

func (br *binaryReader) Int(name string) (int64, bool) {
	v, err := br.r.ReadVarint()
	if err != nil {
		return 0, false
	}
	return v, true
}

func (br *binaryReader) Uint(name string) (uint64, bool) {
	v, err := br.r.ReadUvarint()
	if err != nil {
		return 0, false
	}
	return v, true
}

func (br *binaryReader) Float(name string) (float64, bool) {
	b, err := br.r.Slice(8)
	if err != nil {
		return 0, false
	}
	bits := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	return math.Float64frombits(bits), true
}

func (br *binaryReader) Bool(name string) (bool, bool) {
	b, err := br.r.ReadByte()
	if err != nil {
		return false, false
	}
	return b == 1, true
}

func (br *binaryReader) Bytes(name string) ([]byte, bool) {
	l, err := br.r.ReadUvarint()
	if err != nil {
		return nil, false
	}
	if l == 0 {
		return nil, true
	}
	b, err := br.r.Slice(int(l))
	if err != nil {
		return nil, false
	}
	return b, true
}

func (br *binaryReader) Object(name string, into fmt.Decodable) bool {
	if into == nil {
		return false
	}
	presence, err := br.r.ReadByte()
	if err != nil || presence == 0 {
		return false
	}
	err = into.DecodeFields(br)
	return err == nil
}

func (br *binaryReader) Array(name string) (fmt.ArrayReader, bool) {
	l, err := br.r.ReadUvarint()
	if err != nil {
		return nil, false
	}
	return &binaryArrayReader{br: br, len: int(l)}, true
}

// ArrayReader implementation

type binaryArrayReader struct {
	br  *binaryReader
	len int
}

func (ar *binaryArrayReader) Len() int {
	return ar.len
}

func (ar *binaryArrayReader) String(i int) string {
	val, _ := ar.br.String("")
	return val
}

func (ar *binaryArrayReader) Int(i int) int64 {
	val, _ := ar.br.Int("")
	return val
}

func (ar *binaryArrayReader) Float(i int) float64 {
	val, _ := ar.br.Float("")
	return val
}

func (ar *binaryArrayReader) Bool(i int) bool {
	val, _ := ar.br.Bool("")
	return val
}

func (ar *binaryArrayReader) Bytes(i int) []byte {
	val, _ := ar.br.Bytes("")
	return val
}

func (ar *binaryArrayReader) Object(i int, into fmt.Decodable) bool {
	return ar.br.Object("", into)
}
