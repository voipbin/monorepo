package pipecatcallhandler

import (
	"bytes"
	"testing"
)

func Test_bufferPool(t *testing.T) {
	// Get a buffer from the pool
	buf := getBuffer()
	if buf == nil {
		t.Fatal("getBuffer returned nil")
	}

	// Write some data
	buf.WriteString("test data")
	if buf.Len() != 9 {
		t.Errorf("expected len 9, got %d", buf.Len())
	}

	// Return to pool
	putBuffer(buf)

	// Get another buffer - should be reset
	buf2 := getBuffer()
	if buf2.Len() != 0 {
		t.Errorf("buffer from pool should be empty, got len %d", buf2.Len())
	}
	putBuffer(buf2)
}

func Benchmark_bufferPoolVsNew(b *testing.B) {
	b.Run("sync.Pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := getBuffer()
			buf.Write(make([]byte, 640)) // typical audio frame
			putBuffer(buf)
		}
	})

	b.Run("new_each_time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.Write(make([]byte, 640))
			_ = buf // prevent optimization
		}
	})
}
