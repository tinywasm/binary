package binary

import (
	"unsafe"
)

// toString converts byte slice to a string without allocating.
func toString(b *[]byte) string {
	return *(*string)(unsafe.Pointer(b))
}

// toBytes converts a string to a byte slice without allocating.
func toBytes(v string) []byte {
	// Use unsafe.StringData to get the data pointer directly
	data := unsafe.StringData(v)
	bytesData := unsafe.Slice(data, len(v))

	return bytesData
}

func binaryToBools(b *[]byte) []bool {
	return *(*[]bool)(unsafe.Pointer(b))
}

func boolsToBinary(v *[]bool) []byte {
	return *(*[]byte)(unsafe.Pointer(v))
}
