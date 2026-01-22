package pipecatcallhandler

import (
	"testing"
)

func Test_upgraderBufferSizes(t *testing.T) {
	// Verify buffer sizes are adequate for audio streaming
	// Audio at 16kHz, 20ms chunks = ~640 bytes + protobuf overhead
	// Minimum recommended: 64KB for read, 64KB for write

	minBufferSize := 64 * 1024 // 64KB

	if upgrader.ReadBufferSize < minBufferSize {
		t.Errorf("ReadBufferSize too small: got %d, want >= %d",
			upgrader.ReadBufferSize, minBufferSize)
	}

	if upgrader.WriteBufferSize < minBufferSize {
		t.Errorf("WriteBufferSize too small: got %d, want >= %d",
			upgrader.WriteBufferSize, minBufferSize)
	}
}
