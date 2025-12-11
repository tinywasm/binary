package gobin

import (
	"unsafe"
)

// ToString converts byte slice to a string without allocating.
func ToString(b *[]byte) string {
	return *(*string)(unsafe.Pointer(b))
}

// ToBytes converts a string to a byte slice without allocating.
func ToBytes(v string) []byte {
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
