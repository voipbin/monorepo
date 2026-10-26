package agenthandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_CountByCustomerID_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newWrapTestHandler(mc)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")

	errBoom := stderrors.New("boom")
	m.db.EXPECT().AgentCountByCustomerID(ctx, customerID).Return(0, errBoom)

	count, err := h.CountByCustomerID(ctx, customerID)
	if !stderrors.Is(err, errBoom) {
		t.Errorf("Wrong error. expect: wrapped boom, got: %v", err)
	}
	if err == nil || err.Error() != "could not get agent count: boom" {
		t.Errorf("Wrong error.\nexpect: could not get agent count: boom\ngot: %v", err)
	}
	if count != 0 {
		t.Errorf("Wrong count. expect: 0, got: %d", count)
	}
}

func Test_Reserve_dbErrors(t *testing.T) {
	agentID := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")
	refID := uuid.FromStringOrNil("0b5f2d8e-79a4-11ec-a7b1-6ba5b7016d83")
	before := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	errBoom := stderrors.New("boom")

	tests := []struct {
		name string

		mockFunc func(ctx context.Context, m *wrapMocks)
		call     func(t *testing.T, ctx context.Context, h *agentHandler) error

		expectErr string
	}{
		{
			name: "Reserve",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentReserve(ctx, agentID, "groupcall", refID).Return(false, errBoom)
			},
			call: func(t *testing.T, ctx context.Context, h *agentHandler) error {
				ok, err := h.Reserve(ctx, agentID, "groupcall", refID)
				if ok {
					t.Errorf("Wrong match. expect: false, got: true")
				}
				return err
			},

			expectErr: "could not reserve the agent: boom",
		},
		{
			name: "ReserveRelease",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentReserveRelease(ctx, agentID, refID).Return(errBoom)
			},
			call: func(t *testing.T, ctx context.Context, h *agentHandler) error {
				return h.ReserveRelease(ctx, agentID, refID)
			},

			expectErr: "could not release the agent reservation: boom",
		},
		{
			name: "ReserveSweep",

			mockFunc: func(ctx context.Context, m *wrapMocks) {
				m.db.EXPECT().AgentReserveSweep(ctx, before).Return(0, errBoom)
			},
			call: func(t *testing.T, ctx context.Context, h *agentHandler) error {
				count, err := h.ReserveSweep(ctx, before)
				if count != 0 {
					t.Errorf("Wrong count. expect: 0, got: %d", count)
				}
				return err
			},

			expectErr: "could not sweep zombie agent reservations: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newWrapTestHandler(mc)
			ctx := context.Background()

			tt.mockFunc(ctx, m)

			err := tt.call(t, ctx, h)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong error.\nexpect: %s\ngot: %v", tt.expectErr, err)
			}
		})
	}
}
