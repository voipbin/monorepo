package pipecatcallhandler

import (
	"bytes"
	"sync"
)

// bufferPool provides reusable bytes.Buffer instances to reduce
// GC pressure in audio processing hot paths.
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// getBuffer retrieves a buffer from the pool.
// The caller must call putBuffer when done.
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer returns a buffer to the pool after resetting it.
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}
