package pipecatcall

import (
	"sync"
	"testing"
)

func TestNewAudioJitterBuffer(t *testing.T) {
	jb := NewAudioJitterBuffer()
	if jb == nil {
		t.Fatal("expected non-nil jitter buffer")
	}
	if jb.Len() != 0 {
		t.Errorf("expected empty buffer, got %d bytes", jb.Len())
	}
}

func TestWriteAndReadChunk(t *testing.T) {
	jb := NewAudioJitterBuffer()

	// Write exactly one chunk worth of data
	data := make([]byte, jitterBufDrainChunkSize)
	for i := range data {
		data[i] = byte(i % 256)
	}
	n := jb.Write(data)
	if n != jitterBufDrainChunkSize {
		t.Errorf("expected write to return %d, got %d", jitterBufDrainChunkSize, n)
	}
	if jb.Len() != jitterBufDrainChunkSize {
		t.Errorf("expected buffer length %d, got %d", jitterBufDrainChunkSize, jb.Len())
	}

	// Read one chunk
	chunk := jb.ReadChunk()
	if chunk == nil {
		t.Fatal("expected non-nil chunk")
	}
	if len(chunk) != jitterBufDrainChunkSize {
		t.Errorf("expected chunk size %d, got %d", jitterBufDrainChunkSize, len(chunk))
	}
	for i := range chunk {
		if chunk[i] != data[i] {
			t.Errorf("byte mismatch at index %d: expected %d, got %d", i, data[i], chunk[i])
			break
		}
	}

	// Buffer should be empty now
	if jb.Len() != 0 {
		t.Errorf("expected empty buffer after read, got %d bytes", jb.Len())
	}
}

func TestReadChunkInsufficientData(t *testing.T) {
	jb := NewAudioJitterBuffer()

	// Write less than one chunk
	jb.Write(make([]byte, jitterBufDrainChunkSize-1))

	chunk := jb.ReadChunk()
	if chunk != nil {
		t.Errorf("expected nil chunk when insufficient data, got %d bytes", len(chunk))
	}

	// Empty buffer
	chunk = NewAudioJitterBuffer().ReadChunk()
	if chunk != nil {
		t.Error("expected nil chunk from empty buffer")
	}
}

func TestOverflowDiscardsOldest(t *testing.T) {
	jb := NewAudioJitterBuffer()

	// Fill to max
	old := make([]byte, jitterBufMaxBytes)
	for i := range old {
		old[i] = 0xAA
	}
	jb.Write(old)
	if jb.Len() != jitterBufMaxBytes {
		t.Fatalf("expected buffer at max %d, got %d", jitterBufMaxBytes, jb.Len())
	}

	// Write one more chunk — oldest bytes should be discarded
	extra := make([]byte, jitterBufDrainChunkSize)
	for i := range extra {
		extra[i] = 0xBB
	}
	jb.Write(extra)

	if jb.Len() != jitterBufMaxBytes {
		t.Errorf("expected buffer capped at %d, got %d", jitterBufMaxBytes, jb.Len())
	}

	// The last jitterBufDrainChunkSize bytes should be 0xBB
	// Drain until we get to the last chunk
	var lastChunk []byte
	for {
		c := jb.ReadChunk()
		if c == nil {
			break
		}
		lastChunk = c
	}
	if lastChunk == nil {
		t.Fatal("expected at least one chunk")
	}
	for i, b := range lastChunk {
		if b != 0xBB {
			t.Errorf("expected 0xBB at index %d of last chunk, got 0x%02X", i, b)
			break
		}
	}
}

func TestReset(t *testing.T) {
	jb := NewAudioJitterBuffer()
	jb.Write(make([]byte, jitterBufDrainChunkSize*3))

	if jb.Len() == 0 {
		t.Fatal("expected non-empty buffer before reset")
	}

	jb.Reset()
	if jb.Len() != 0 {
		t.Errorf("expected empty buffer after reset, got %d bytes", jb.Len())
	}
	if chunk := jb.ReadChunk(); chunk != nil {
		t.Error("expected nil chunk after reset")
	}
}

func TestConcurrentWriteRead(t *testing.T) {
	jb := NewAudioJitterBuffer()
	const goroutines = 10
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Writers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			data := make([]byte, 160) // 5ms chunk
			for j := 0; j < iterations; j++ {
				jb.Write(data)
			}
		}()
	}

	// Readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				jb.ReadChunk()
			}
		}()
	}

	wg.Wait()

	// Buffer should not exceed max
	if jb.Len() > jitterBufMaxBytes {
		t.Errorf("buffer exceeded max: %d > %d", jb.Len(), jitterBufMaxBytes)
	}
}

func TestMultipleChunksInSequence(t *testing.T) {
	jb := NewAudioJitterBuffer()

	// Write 3 chunks worth of data
	data := make([]byte, jitterBufDrainChunkSize*3)
	for i := range data {
		data[i] = byte(i / jitterBufDrainChunkSize)
	}
	jb.Write(data)

	// Read 3 chunks, verify each has the expected fill byte
	for chunkIdx := 0; chunkIdx < 3; chunkIdx++ {
		chunk := jb.ReadChunk()
		if chunk == nil {
			t.Fatalf("expected chunk %d, got nil", chunkIdx)
		}
		expected := byte(chunkIdx)
		if chunk[0] != expected {
			t.Errorf("chunk %d: expected first byte %d, got %d", chunkIdx, expected, chunk[0])
		}
	}

	// Fourth read should return nil
	if chunk := jb.ReadChunk(); chunk != nil {
		t.Error("expected nil after draining all chunks")
	}
}
