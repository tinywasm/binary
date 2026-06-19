package binary

import (
	"bytes"
	"io"
	"sync"

	"github.com/tinywasm/fmt"
)

var (
	writerPool = sync.Pool{
		New: func() any {
			return newWriter(nil)
		},
	}
	readerPool = sync.Pool{
		New: func() any {
			return newBinaryReader(nil)
		},
	}
)

// Encode encodes input to output.
// input: Encodable struct
// output: *[]byte or io.Writer
func Encode(input fmt.Encodable, output any) error {
	if input == nil || input.IsNil() {
		return fmt.Err("Encode: input is nil")
	}

	w := writerPool.Get().(*binaryWriter)
	defer writerPool.Put(w)

	var err error
	switch out := output.(type) {
	case *[]byte:
		var buffer bytes.Buffer
		w.reset(&buffer)
		input.EncodeFields(w)
		if w.err == nil {
			*out = buffer.Bytes()
		}
		err = w.err
	case io.Writer:
		w.reset(out)
		input.EncodeFields(w)
		err = w.err
	default:
		err = fmt.Err("Encode", "output", "must be *[]byte or io.Writer")
	}

	return err
}

// Decode decodes input to output.
// input: []byte or io.Reader
// output: pointer to Decodable struct
func Decode(input, output any) error {
	if output == nil {
		return fmt.Err("Decode: output is nil")
	}

	dec, ok := output.(fmt.Decodable)
	if !ok {
		return fmt.Err("Decode", "output", "must implement fmt.Decodable")
	}
	if dec.IsNil() {
		return fmt.Err("Decode: output is nil")
	}

	r := readerPool.Get().(*binaryReader)
	defer readerPool.Put(r)

	var err error
	switch in := input.(type) {
	case []byte:
		r.reset(bytes.NewReader(in))
		err = dec.DecodeFields(r)
	case io.Reader:
		r.reset(in)
		err = dec.DecodeFields(r)
	default:
		err = fmt.Err("Decode", "input", "must be []byte or io.Reader")
	}

	return err
}

// SetLog is deprecated and does nothing.
func SetLog(fn func(msg ...any)) {}

// Errorf is a helper for fmt.Errorf
func Errorf(format string, a ...any) error {
	return fmt.Errf(format, a...)
}
