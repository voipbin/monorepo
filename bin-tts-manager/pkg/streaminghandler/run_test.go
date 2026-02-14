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
