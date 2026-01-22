package pipecatcallhandler

import (
	"testing"
	"time"
)

func Test_defaultPushFrameTimeout(t *testing.T) {
	// Frame timeout should be < 100ms for real-time audio
	// 2 seconds is far too long and causes unacceptable latency

	maxAcceptableTimeout := 100 * time.Millisecond

	if defaultPushFrameTimeout > maxAcceptableTimeout {
		t.Errorf("defaultPushFrameTimeout too high: got %v, want <= %v",
			defaultPushFrameTimeout, maxAcceptableTimeout)
	}
}

func Test_defaultRunnerWebsocketChanBufferSize(t *testing.T) {
	// Buffer size should be 100-200 frames (2-4 seconds at 50fps)
	// 2000 frames = 40 seconds of buffering, causes memory bloat

	maxAcceptableSize := 200

	if defaultRunnerWebsocketChanBufferSize > maxAcceptableSize {
		t.Errorf("defaultRunnerWebsocketChanBufferSize too high: got %d, want <= %d",
			defaultRunnerWebsocketChanBufferSize, maxAcceptableSize)
	}
}
