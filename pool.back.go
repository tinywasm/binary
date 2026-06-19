//go:build !wasm

package binary

import "sync"

var (
	writerPool = sync.Pool{New: func() any { return newWriter(nil) }}
	readerPool = sync.Pool{New: func() any { return newBinaryReader(nil) }}
)

func getWriter() *binaryWriter  { return writerPool.Get().(*binaryWriter) }
func putWriter(w *binaryWriter) { writerPool.Put(w) }

func getReader() *binaryReader  { return readerPool.Get().(*binaryReader) }
func putReader(r *binaryReader) { readerPool.Put(r) }
