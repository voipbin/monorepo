package streaminghandler

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gofrs/uuid"
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

func Test_runKeepAlive(t *testing.T) {
	tests := []struct {
		name        string
		interval    time.Duration
		streamingID uuid.UUID

		expectWrites  int
		cancelAfterMs int
	}{
		{
			name:          "normal",
			interval:      300 * time.Millisecond,
			streamingID:   uuid.FromStringOrNil("eee08956-3bd7-11f0-b6f0-9fd777b76ce0"),
			expectWrites:  3,
			cancelAfterMs: 1050,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockConn := new(MockConn)
			mockConn.On("Write", []byte{0x10, 0x00, 0x01, 0x00}).Return(4, nil)

			ctx, cancel := context.WithCancel(context.Background())

			defer func() {
				cancel()
				mockConn.AssertExpectations(t)
			}()

			go func() {
				time.Sleep(time.Duration(tt.cancelAfterMs) * time.Millisecond)
				cancel()
			}()

			handler := &streamingHandler{}
			handler.runKeepAlive(ctx, cancel, mockConn, tt.interval, tt.streamingID)

			mockConn.AssertNumberOfCalls(t, "Write", tt.expectWrites)
		})
	}
}
