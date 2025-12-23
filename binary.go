package binary

import (
	"bytes"
	"io"
	"reflect"
	"sync"

	. "github.com/tinywasm/fmt"
)

var (
	global *instance
	once   sync.Once
)

func getInstance() *instance {
	once.Do(func() {
		global = newInstance()
	})
	return global
}

// Encode encodes input to output.
// input: struct or pointer to struct
// output: *[]byte or io.Writer
func Encode(input, output any) error {
	inst := getInstance()

	switch out := output.(type) {
	case *[]byte:
		var buffer bytes.Buffer
		buffer.Grow(64)
		if err := inst.encodeTo(input, &buffer); err == nil {
			*out = buffer.Bytes()
			return nil
		} else {
			return err
		}
	case io.Writer:
		return inst.encodeTo(input, out)
	default:
		return Err("Encode", "output", "must be *[]byte or io.Writer")
	}
}

// Decode decodes input to output.
// input: []byte or io.Reader
// output: pointer to struct
func Decode(input, output any) error {
	inst := getInstance()

	switch in := input.(type) {
	case []byte:
		return inst.decode(in, output)
	case io.Reader:
		return inst.decodeFrom(in, output)
	default:
		return Err("Decode", "input", "must be []byte or io.Reader")
	}
}

// SetLog sets a custom logging function for debug/testing.
// Pass nil to disable logging.
func SetLog(fn func(msg ...any)) {
	getInstance().log = fn
}

// instance represents a binary encoder/decoder with isolated state.
type instance struct {
	// log is an optional custom logging function
	log func(msg ...any)

	// schemas is a slice-based cache for TinyGo compatibility (no maps allowed)
	schemas []schemaEntry

	// encoders is a private pool for encoder instances
	encoders *sync.Pool

	// decoders is a private pool for decoder instances
	decoders *sync.Pool

	// Mutex to protect schemas slice
	mu sync.RWMutex
}

// schemaEntry represents a cached schema with its type and codec
type schemaEntry struct {
	Type  reflect.Type
	codec codec
}

func newInstance(args ...any) *instance {
	var logFunc func(msg ...any) // Default: no logging

	for _, arg := range args {
		if log, ok := arg.(func(msg ...any)); ok {
			logFunc = log
		}
	}

	tb := &instance{log: logFunc}

	tb.schemas = make([]schemaEntry, 0, 100) // Pre-allocate reasonable size
	tb.encoders = &sync.Pool{
		New: func() any {
			return &encoder{
				tb: tb,
			}
		},
	}
	tb.decoders = &sync.Pool{
		New: func() any {
			return &decoder{
				tb: tb,
			}
		},
	}

	return tb
}

// EncodeTo encodes the payload into a specific destination using this instance.
func (tb *instance) encodeTo(data any, dst io.Writer) error {
	// Get the encoder from the pool, reset it
	e := tb.encoders.Get().(*encoder)
	e.reset(dst, tb)

	// Encode and set the buffer if successful
	err := e.encode(data)

	// Put the encoder back when we're finished
	tb.encoders.Put(e)
	return err
}

// Decode decodes the payload from the binary format using this instance.
func (tb *instance) decode(data []byte, target any) error {
	// Get the decoder from the pool, reset it
	d := tb.decoders.Get().(*decoder)
	d.reset(data, tb)

	// Decode and free the decoder
	err := d.decode(target)
	tb.decoders.Put(d)
	return err
}

func (tb *instance) decodeFrom(r io.Reader, target any) error {
	// Get the decoder from the pool, reset it
	d := tb.decoders.Get().(*decoder)
	if d.reader == nil {
		d.reader = newReader(r)
	} else {
		// If it's a slice reader, we need to replace it with a stream reader
		if _, ok := d.reader.(*sliceReader); ok {
			d.reader = newReader(r)
		} else {
			// It's already a stream reader, but we can't easily reset its inner reader
			// because streamReader has a private field.
			// Let's just create a new reader for now or check if we can reset it.
			d.reader = newReader(r)
		}
	}
	d.tb = tb

	// Decode and free the decoder
	err := d.decode(target)
	tb.decoders.Put(d)
	return err
}

// findSchema performs a linear search in the slice-based cache for TinyGo compatibility
func (tb *instance) findSchema(t reflect.Type) (codec, bool) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	for _, entry := range tb.schemas {
		if entry.Type == t {
			return entry.codec, true
		}
	}
	return nil, false
}

// addSchema adds a new schema to the slice-based cache
func (tb *instance) addSchema(t reflect.Type, codec codec) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	// Simple cache size limit (optional, for memory control)
	if len(tb.schemas) >= 1000 { // Reasonable default limit
		// Simple eviction: remove oldest (first) entry
		tb.schemas = tb.schemas[1:]
	}

	tb.schemas = append(tb.schemas, schemaEntry{
		Type:  t,
		codec: codec,
	})
}

// scanToCache scans the type and caches it in the instance using slice-based cache
func (tb *instance) scanToCache(t reflect.Type) (codec, error) {
	if t == nil {
		return nil, Err("scanToCache", "type", "nil")
	}

	// Check if we already have this schema cached
	if c, found := tb.findSchema(t); found {
		return c, nil
	}

	// Scan for the first time
	c, err := scan(t)
	if err != nil {
		return nil, err
	}

	// Cache the schema
	tb.addSchema(t, c)

	return c, nil
}
