package dbhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_CallApplicationAMDGet(t *testing.T) {
	tests := []struct {
		name        string
		channelID   string
		cacheReturn *callapplication.AMD
		cacheErr    error
		expectErr   bool
	}{
		{
			name:      "successful get",
			channelID: "test-channel-1",
			cacheReturn: &callapplication.AMD{
				CallID:        uuid.Must(uuid.NewV4()),
				MachineHandle: "continue",
				Async:         true,
			},
			cacheErr:  nil,
			expectErr: false,
		},
		{
			name:        "cache error",
			channelID:   "test-channel-2",
			cacheReturn: nil,
			cacheErr:    fmt.Errorf("cache error"),
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().
				CallAppAMDGet(ctx, tt.channelID).
				Return(tt.cacheReturn, tt.cacheErr)

			res, err := h.CallApplicationAMDGet(ctx, tt.channelID)
			if (err != nil) != tt.expectErr {
				t.Errorf("CallApplicationAMDGet() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && res != tt.cacheReturn {
				t.Errorf("CallApplicationAMDGet() = %v, want %v", res, tt.cacheReturn)
			}
		})
	}
}

func Test_CallApplicationAMDSet(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		app       *callapplication.AMD
		cacheErr  error
		expectErr bool
	}{
		{
			name:      "successful set",
			channelID: "test-channel-1",
			app: &callapplication.AMD{
				CallID:        uuid.Must(uuid.NewV4()),
				MachineHandle: "continue",
				Async:         true,
			},
			cacheErr:  nil,
			expectErr: false,
		},
		{
			name:      "cache error",
			channelID: "test-channel-2",
			app: &callapplication.AMD{
				CallID:        uuid.Must(uuid.NewV4()),
				MachineHandle: "hangup",
				Async:         false,
			},
			cacheErr:  fmt.Errorf("cache error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().
				CallAppAMDSet(ctx, tt.channelID, tt.app).
				Return(tt.cacheErr)

			err := h.CallApplicationAMDSet(ctx, tt.channelID, tt.app)
			if (err != nil) != tt.expectErr {
				t.Errorf("CallApplicationAMDSet() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

