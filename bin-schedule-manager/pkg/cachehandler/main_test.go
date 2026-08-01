package cachehandler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-redsync/redsync/v4"
	"github.com/gofrs/uuid"
)

func Test_NewHandler(t *testing.T) {
	h := NewHandler("127.0.0.1:6379", "", 1)
	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func Test_scheduleKey(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		expectRes string
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110002"),

			expectRes: "schedule:5f9c1b2a-6f13-11f0-c001-0242ac110002",
		},
		{
			name: "nil uuid",
			id:   uuid.Nil,

			expectRes: "schedule:00000000-0000-0000-0000-000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := scheduleKey(tt.id); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_scheduleLockKey(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		expectRes string
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("5f9c1b2a-6f13-11f0-c001-0242ac110002"),

			expectRes: "schedule:lock:5f9c1b2a-6f13-11f0-c001-0242ac110002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := scheduleLockKey(tt.id); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_isLockBusyError(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expectRes bool
	}{
		{
			name: "redsync ErrFailed is busy",
			err:  redsync.ErrFailed,

			expectRes: true,
		},
		{
			name: "redsync ErrTaken is busy",
			err:  &redsync.ErrTaken{Nodes: []int{0}},

			expectRes: true,
		},
		{
			name: "redsync ErrNodeTaken is busy",
			err:  &redsync.ErrNodeTaken{Node: 0},

			expectRes: true,
		},
		{
			name: "wrapped ErrFailed is busy",
			err:  fmt.Errorf("wrapped: %w", redsync.ErrFailed),

			expectRes: true,
		},
		{
			name: "other error is not busy",
			err:  errors.New("connection refused"),

			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isLockBusyError(tt.err); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ErrLockBusy(t *testing.T) {
	if ErrLockBusy == nil {
		t.Errorf("Expected sentinel error, got nil")
	}
}
