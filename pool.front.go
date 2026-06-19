//go:build wasm

package binary

// WASM is single-threaded: allocate directly instead of sync.Pool.

func getWriter() *binaryWriter { return newWriter(nil) }
func putWriter(_ *binaryWriter) {}

func getReader() *binaryReader { return newBinaryReader(nil) }
func putReader(_ *binaryReader) {}
