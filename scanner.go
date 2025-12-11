package gobin

import (
	"reflect"

	. "github.com/tinywasm/fmt"
)

// Note: Global schemas map removed - now using instance-based caching in GoBin

// scanToCache scans the type and caches in the local cache.
func scanToCache(t reflect.Type, cache map[reflect.Type]Codec) (Codec, error) {
	if c, ok := cache[t]; ok {
		return c, nil
	}

	c, err := scan(t)
	if err != nil {
		return nil, err
	}

	cache[t] = c
	return c, nil
}

// Scan gets a codec for the type. Caching is now handled by GoBin instance.
func scan(t reflect.Type) (c Codec, err error) {
	return scanType(t)
}

// ScanType scans the type
func scanType(t reflect.Type) (Codec, error) {
	if t == nil {
		return nil, Err(D.Value, D.Type, D.Nil)
	}

	// Check if the type or a pointer to it implements the marshaling interfaces.
	pt := reflect.PtrTo(t)
	if t.Implements(binaryMarshalerType) && pt.Implements(binaryUnmarshalerType) {
		return new(binaryMarshalerCodec), nil
	}
	if pt.Implements(binaryMarshalerType) && pt.Implements(binaryUnmarshalerType) {
		return new(binaryMarshalerCodec), nil
	}

	// TODO: Implement custom codec scanning when needed
	// if custom, ok := scanCustomCodec(t); ok {
	//     return custom, nil
	// }

	// TODO: Implement binary marshaler scanning when needed
	// if custom, ok := scanBinaryMarshaler(t); ok {
	//     return custom, nil
	// }

	switch t.Kind() {
	case reflect.Ptr:
		elem := t.Elem()
		elemCodec, err := scanType(elem)
		if err != nil {
			return nil, err
		}

		return &reflectPointerCodec{
			elemCodec: elemCodec,
		}, nil

	case reflect.Array:
		elem := t.Elem()
		elemCodec, err := scanType(elem)
		if err != nil {
			return nil, err
		}

		return &reflectArrayCodec{
			elemCodec: elemCodec,
		}, nil

	case reflect.Slice:
		elem := t.Elem()
		elemKind := elem.Kind()

		// Fast-paths for simple numeric slices and string slices
		switch elemKind {
		case reflect.Uint8:
			return new(byteSliceCodec), nil
		case reflect.Bool:
			return new(boolSliceCodec), nil
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return new(varuintSliceCodec), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return new(varintSliceCodec), nil
		case reflect.Ptr:
			elemElem := elem.Elem()
			elemCodec, err := scanType(elemElem)
			if err != nil {
				return nil, err
			}

			return &reflectSliceOfPtrCodec{
				elemType:  elemElem,
				elemCodec: elemCodec,
			}, nil
		default:
			elemCodec, err := scanType(elem)
			if err != nil {
				return nil, err
			}

			return &reflectSliceCodec{
				elemCodec: elemCodec,
			}, nil
		}

	case reflect.Struct:
		s := scanStruct(t)
		v := make(reflectStructCodec, 0, len(s.fields))
		for _, i := range s.fields {
			field := t.Field(i)
			codec, err := scanType(field.Type)
			if err != nil {
				return nil, err
			}

			// Append since unexported fields are skipped
			v = append(v, fieldCodec{
				Index: i,
				Codec: codec,
			})
		}

		return &v, nil

	case reflect.String:
		return new(stringCodec), nil
	case reflect.Bool:
		return new(boolCodec), nil
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64:
		return new(varintCodec), nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint, reflect.Uint64:
		return new(varuintCodec), nil
	case reflect.Float32:
		return new(float32Codec), nil
	case reflect.Float64:
		return new(float64Codec), nil
	}

	return nil, Err(D.Type, D.Binary, t.String(), D.Not, D.Supported)
}

type scannedStruct struct {
	fields []int
}

// scanStruct scans a struct using reflect.Type
func scanStruct(t reflect.Type) *scannedStruct {
	numFields := t.NumField()
	meta := &scannedStruct{fields: make([]int, 0, numFields)}
	for i := 0; i < numFields; i++ {
		field := t.Field(i)

		// Get field name
		if field.Name != "_" {
			// Check if field should be skipped
			tag := field.Tag
			if tag.Get("binary") != "-" {
				meta.fields = append(meta.fields, i)
			}
		}
	}
	return meta
}
