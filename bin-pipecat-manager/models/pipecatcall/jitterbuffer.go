package pipecatcall

import "sync"

const (
	// jitterBufMaxBytes is the maximum number of bytes the jitter buffer will
	// hold. Beyond this limit the oldest data is discarded. 500ms of 16kHz
	// mono 16-bit PCM = 16000 bytes.
	jitterBufMaxBytes = 16000

	// jitterBufDrainChunkSize is the number of bytes drained per tick. 20ms
	// of 16kHz mono 16-bit PCM = 640 bytes — matching Asterisk's mixing timer.
	jitterBufDrainChunkSize = 640
)

// AudioJitterBuffer is a simple byte-level ring buffer that absorbs timing
// irregularities from the Python pipecat runner and drains at a fixed 20ms
// cadence toward Asterisk.
type AudioJitterBuffer struct {
	mu  sync.Mutex
	buf []byte
}

// NewAudioJitterBuffer returns a ready-to-use jitter buffer.
func NewAudioJitterBuffer() *AudioJitterBuffer {
	return &AudioJitterBuffer{}
}

// Write appends data to the buffer. If the buffer would exceed
// jitterBufMaxBytes, the oldest bytes are discarded to make room.
// Returns the number of bytes accepted (always len(data)).
func (jb *AudioJitterBuffer) Write(data []byte) int {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	jb.buf = append(jb.buf, data...)

	if len(jb.buf) > jitterBufMaxBytes {
		jb.buf = jb.buf[len(jb.buf)-jitterBufMaxBytes:]
	}

	return len(data)
}

// ReadChunk returns exactly jitterBufDrainChunkSize bytes from the front of
// the buffer, or nil if insufficient data is available.
func (jb *AudioJitterBuffer) ReadChunk() []byte {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.buf) < jitterBufDrainChunkSize {
		return nil
	}

	chunk := make([]byte, jitterBufDrainChunkSize)
	copy(chunk, jb.buf[:jitterBufDrainChunkSize])
	jb.buf = jb.buf[jitterBufDrainChunkSize:]

	return chunk
}

// Len returns the current number of buffered bytes.
func (jb *AudioJitterBuffer) Len() int {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	return len(jb.buf)
}

// Reset clears the buffer.
func (jb *AudioJitterBuffer) Reset() {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	jb.buf = nil
}
