# TTS AudioSocket Silence Feed Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix "use of closed network connection" error when speaking/say requests are made, caused by Asterisk tearing down the AudioSocket channel because no audio frames are sent during TTS initialization.

**Architecture:** Replace the 10-second keepalive with a continuous 20ms silence feed that sends proper AudioSocket frames matching Asterisk's media loop timing. This prevents Asterisk's `audiosocket_read` from getting EAGAIN and killing the channel.

**Tech Stack:** Go, AudioSocket protocol, Asterisk external media

---

### Task 1: Write failing test for runSilenceFeed

**Files:**
- Modify: `bin-tts-manager/pkg/streaminghandler/run_test.go`

**Step 1: Write the test**

Replace the existing `Test_runKeepAlive` test with `Test_runSilenceFeed`. The new test verifies that `runSilenceFeed` sends properly formatted 20ms silence frames (320 bytes of zeros wrapped in AudioSocket format) at 20ms intervals.

The expected frame written to the connection is the output of `audiosocketWrapDataPCM16Bit(make([]byte, 320))`:
- 1 byte: format `0x10`
- 2 bytes: payload length `0x01, 0x40` (320 in big-endian)
- 320 bytes: zeros

```go
package streaminghandler

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockConn is a mock implementation of the net.Conn interface
type MockConn struct {
	mock.Mock
}

func (m *MockConn) Write(b []byte) (int, error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}

func (m *MockConn) Close() error {
	return m.Called().Error(0)
}

func (m *MockConn) Read(b []byte) (int, error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}

func (m *MockConn) LocalAddr() net.Addr {
	return m.Called().Get(0).(net.Addr)
}

func (m *MockConn) RemoteAddr() net.Addr {
	return m.Called().Get(0).(net.Addr)
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return m.Called(t).Error(0)
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return m.Called(t).Error(0)
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return m.Called(t).Error(0)
}

func Test_runSilenceFeed(t *testing.T) {
	// Build expected silence frame: audiosocketWrapDataPCM16Bit(make([]byte, 320))
	expectedFrame, err := audiosocketWrapDataPCM16Bit(make([]byte, audiosocketSilenceFrameSize))
	if err != nil {
		t.Fatalf("Failed to build expected silence frame: %v", err)
	}

	tests := []struct {
		name string

		cancelAfter  time.Duration
		expectWrites int
	}{
		{
			name:         "sends multiple silence frames before cancel",
			cancelAfter:  250 * time.Millisecond,
			expectWrites: 12, // ~250ms / 20ms = 12.5, expect ~12
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockConn := new(MockConn)
			mockConn.On("Write", expectedFrame).Return(len(expectedFrame), nil)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				time.Sleep(tt.cancelAfter)
				cancel()
			}()

			handler := &streamingHandler{}
			handler.runSilenceFeed(ctx, cancel, mockConn)

			calls := len(mockConn.Calls)
			if calls < tt.expectWrites-3 || calls > tt.expectWrites+3 {
				t.Errorf("Expected approximately %d writes, got %d", tt.expectWrites, calls)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Fix-tts-audiosocket-silence-feed/bin-tts-manager && go test -v -run Test_runSilenceFeed ./pkg/streaminghandler/...`
Expected: FAIL — `runSilenceFeed` method does not exist, `audiosocketSilenceFrameSize` undefined

---

### Task 2: Implement runSilenceFeed and update runStart

**Files:**
- Modify: `bin-tts-manager/pkg/streaminghandler/run.go` (replace `runKeepAlive` and `retryWithBackoff` with `runSilenceFeed`)
- Modify: `bin-tts-manager/pkg/streaminghandler/main.go:66-70` (replace constants)

**Step 1: Update constants in main.go**

Replace lines 66-70 in `main.go`:

```go
// Old:
const (
	defaultKeepAliveInterval = 10 * time.Second // 10 seconds
	defaultMaxRetryAttempts  = 3
	defaultInitialBackoff    = 100 * time.Millisecond // 100 milliseconds
)

// New:
const (
	defaultSilenceFeedInterval = 20 * time.Millisecond // 20ms matches Asterisk's media loop timing
)
```

**Step 2: Replace runKeepAlive with runSilenceFeed in run.go**

Delete `runKeepAlive` (lines 97-127) and `retryWithBackoff` (lines 129-144).

Add `runSilenceFeed`:

```go
// runSilenceFeed sends 20ms silence frames to the Asterisk AudioSocket connection
// at regular intervals. This prevents Asterisk's audiosocket_read from getting EAGAIN
// (Resource temporarily unavailable) and tearing down the channel.
//
// Asterisk's bridge media loop reads audio frames every ~20ms. If no data is available,
// res_audiosocket.c returns an error and chan_audiosocket.c hangs up the channel.
// This function keeps the connection alive by sending silence (zero-filled PCM frames).
func (h *streamingHandler) runSilenceFeed(ctx context.Context, cancel context.CancelFunc, conn net.Conn) {
	log := logrus.WithField("func", "runSilenceFeed")
	defer cancel()

	silenceData := make([]byte, audiosocketSilenceFrameSize)
	silenceFrame, err := audiosocketWrapDataPCM16Bit(silenceData)
	if err != nil {
		log.Errorf("Failed to create silence frame: %v", err)
		return
	}

	ticker := time.NewTicker(defaultSilenceFeedInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("Silence feed stopped")
			return

		case <-ticker.C:
			if _, errWrite := conn.Write(silenceFrame); errWrite != nil {
				log.Errorf("Failed to send silence frame: %v", errWrite)
				return
			}
		}
	}
}
```

**Step 3: Add the silence frame size constant to audiosocket.go**

Add after the existing constants (line 21 in audiosocket.go):

```go
audiosocketSilenceFrameSize = 320 // 160 samples * 2 bytes/sample = 20ms at 8kHz 16-bit mono
```

**Step 4: Update runStart to call runSilenceFeed instead of runKeepAlive**

In `runStart`, replace line 58:

```go
// Old:
go h.runKeepAlive(ctx, cancel, conn, defaultKeepAliveInterval, streamingID)

// New:
go h.runSilenceFeed(ctx, cancel, conn)
```

**Step 5: Remove unused uuid import from run.go**

After removing `runKeepAlive`, the `uuid` import is unused. Remove `"github.com/gofrs/uuid"` from the import block. Also remove `"time"` since `runSilenceFeed` gets its interval from the constant (which is in main.go, same package).

Actually — `time` IS still needed because `time.NewTicker` is used in `runSilenceFeed`. Keep `"time"`. Remove only `"github.com/gofrs/uuid"`.

**Step 6: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Fix-tts-audiosocket-silence-feed/bin-tts-manager && go test -v -run Test_runSilenceFeed ./pkg/streaminghandler/...`
Expected: PASS

---

### Task 3: Run full verification workflow

**Step 1: Run the full verification for bin-tts-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Fix-tts-audiosocket-silence-feed/bin-tts-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass. If linting catches unused variables/imports from the removal, fix them.

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Fix-tts-audiosocket-silence-feed
git add bin-tts-manager/pkg/streaminghandler/run.go \
        bin-tts-manager/pkg/streaminghandler/run_test.go \
        bin-tts-manager/pkg/streaminghandler/main.go \
        bin-tts-manager/pkg/streaminghandler/audiosocket.go
git commit -m "NOJIRA-Fix-tts-audiosocket-silence-feed

Replace 10-second keepalive with continuous 20ms silence feed to prevent
Asterisk from tearing down AudioSocket channel due to EAGAIN on read.

- bin-tts-manager: Add runSilenceFeed that sends 320-byte silence frames at 20ms intervals
- bin-tts-manager: Remove runKeepAlive and retryWithBackoff (replaced by silence feed)
- bin-tts-manager: Add audiosocketSilenceFrameSize constant (320 bytes = 20ms at 8kHz)
- bin-tts-manager: Update runStart to use runSilenceFeed instead of runKeepAlive"
```
