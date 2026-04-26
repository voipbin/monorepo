package aicallhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	pkgerrors "github.com/pkg/errors"
	gomock "go.uber.org/mock/gomock"
)

func Test_aicallHandler_pingPipecatHost(t *testing.T) {
	tests := []struct {
		name string

		hostID    string
		mockErr   error
		expectRes bool
		expectPing bool
	}{
		{
			name: "empty host_id returns false without calling ping",

			hostID:     "",
			expectRes:  false,
			expectPing: false,
		},
		{
			name: "alive pod returns true",

			hostID:     "10.4.2.18",
			mockErr:    nil,
			expectRes:  true,
			expectPing: true,
		},
		{
			name: "wrapped DeadlineExceeded returns false",

			hostID:     "10.4.2.19",
			mockErr:    pkgerrors.Wrap(context.DeadlineExceeded, "rpc timeout"),
			expectRes:  false,
			expectPing: true,
		},
		{
			name: "wrapped ErrCircuitOpen returns false",

			hostID:     "10.4.2.20",
			mockErr:    pkgerrors.Wrap(circuitbreakerhandler.ErrCircuitOpen, "ping rejected"),
			expectRes:  false,
			expectPing: true,
		},
		{
			name: "unexpected broker error returns false",

			hostID:     "10.4.2.21",
			mockErr:    fmt.Errorf("broker connection refused"),
			expectRes:  false,
			expectPing: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &aicallHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			if tt.expectPing {
				mockReq.EXPECT().PipecatV1Ping(gomock.Any(), tt.hostID).Return(tt.mockErr)
			}

			got := h.pingPipecatHost(ctx, tt.hostID)
			if got != tt.expectRes {
				t.Errorf("expected: %v, got: %v", tt.expectRes, got)
			}
		})
	}
}
