package binary

import (
	"reflect"

	. "github.com/tinywasm/fmt"
)

// Note: Global schemas map removed - now using instance-based caching in Binary

// Scan gets a codec for the type. Caching is now handled by Binary instance.
func scan(t reflect.Type) (c codec, err error) {
	return scanType(t)
}

// ScanType scans the type
func scanType(t reflect.Type) (codec, error) {
	if t == nil {
		return nil, Err("value", "type", "nil")
	}

	// Check if the type or a pointer to it implements the marshaling interfaces.
	pt := reflect.PointerTo(t)
	if t.Implements(binaryMarshalerType) && pt.Implements(binaryUnmarshalerType) {
		return new(binaryMarshalercodec), nil
	}
	if pt.Implements(binaryMarshalerType) && pt.Implements(binaryUnmarshalerType) {
		return new(binaryMarshalercodec), nil
	}

	// TODO: Implement custom codec scanning when needed
	// if custom, ok := scanCustomcodec(t); ok {
	//     return custom, nil
	// }

	// TODO: Implement binary marshaler scanning when needed
	// if custom, ok := scanBinaryMarshaler(t); ok {
	//     return custom, nil
	// }

	switch t.Kind() {
	case reflect.Ptr:
		elem := t.Elem()
		elemcodec, err := scanType(elem)
		if err != nil {
			return nil, err
		}

		return &reflectPointercodec{
			elemcodec: elemcodec,
		}, nil

	case reflect.Array:
		elem := t.Elem()
		elemcodec, err := scanType(elem)
		if err != nil {
			return nil, err
		}

		return &reflectArraycodec{
			elemcodec: elemcodec,
		}, nil

	case reflect.Slice:
		elem := t.Elem()
		elemKind := elem.Kind()

		// Fast-paths for simple numeric slices and string slices
		switch elemKind {
		case reflect.Uint8:
			return new(byteSlicecodec), nil
		case reflect.Bool:
			return new(boolSlicecodec), nil
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return &numericSlicecodec{signed: false}, nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return &numericSlicecodec{signed: true}, nil
		case reflect.Ptr:
			elemElem := elem.Elem()
			elemcodec, err := scanType(elemElem)
			if err != nil {
				return nil, err
			}

			return &reflectSliceOfPtrcodec{
				elemType:  elemElem,
				elemcodec: elemcodec,
			}, nil
		default:
			elemcodec, err := scanType(elem)
			if err != nil {
				return nil, err
			}

			return &reflectSlicecodec{
				elemcodec: elemcodec,
			}, nil
		}

	case reflect.Struct:
		s := scanStruct(t)
		v := make(reflectStructcodec, 0, len(s.fields))
		for _, i := range s.fields {
			field := t.Field(i)
			codec, err := scanType(field.Type)
			if err != nil {
				return nil, err
			}

			// Append since unexported fields are skipped
			v = append(v, fieldcodec{
				Index: i,
				codec: codec,
			})
		}

		return &v, nil

	case reflect.String:
		return new(stringcodec), nil
	case reflect.Bool:
		return new(boolcodec), nil
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64:
		return new(varintcodec), nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint, reflect.Uint64:
		return new(varuintcodec), nil
	case reflect.Float32:
		return new(float32codec), nil
	case reflect.Float64:
		return new(float64codec), nil
	case reflect.Map:
		keycodec, err := scanType(t.Key())
		if err != nil {
			return nil, err
		}
		valcodec, err := scanType(t.Elem())
		if err != nil {
			return nil, err
		}
		return &mapcodec{
			keycodec:   keycodec,
			valuecodec: valcodec,
		}, nil
	}

	return nil, Err("type", "binary", t.String(), "not", "supported")
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
		if HasUpperPrefix(field.Name) {
			// Check if field should be skipped
			tag := Convert(string(field.Tag))
			jsonTag, _ := tag.TagValue("json")
			binaryTag, _ := tag.TagValue("binary")

			if jsonTag != "-" && binaryTag != "-" {
				meta.fields = append(meta.fields, i)
			}
		}
	}
	return meta
}
