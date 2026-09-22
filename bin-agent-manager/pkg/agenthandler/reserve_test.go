package agenthandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/dbhandler"
)

func Test_Reserve(t *testing.T) {

	tests := []struct {
		name string

		agentID uuid.UUID
		refType string
		refID   uuid.UUID

		responseOK bool
		expectRes  bool
	}{
		{
			name: "reserve wins",

			agentID: uuid.FromStringOrNil("d1000001-1539-0000-0000-000000000001"),
			refType: "queuecall",
			refID:   uuid.FromStringOrNil("d1000001-1539-0000-0000-0000000000a1"),

			responseOK: true,
			expectRes:  true,
		},
		{
			name: "reserve loses",

			agentID: uuid.FromStringOrNil("d1000002-1539-0000-0000-000000000001"),
			refType: "queuecall",
			refID:   uuid.FromStringOrNil("d1000002-1539-0000-0000-0000000000a1"),

			responseOK: false,
			expectRes:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &agentHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AgentReserve(ctx, tt.agentID, tt.refType, tt.refID).Return(tt.responseOK, nil)

			res, err := h.Reserve(ctx, tt.agentID, tt.refType, tt.refID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ReserveRelease(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &agentHandler{
		db: mockDB,
	}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("d1000003-1539-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("d1000003-1539-0000-0000-0000000000a1")

	mockDB.EXPECT().AgentReserveRelease(ctx, agentID, refID).Return(nil)

	if err := h.ReserveRelease(ctx, agentID, refID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ReserveSweep(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &agentHandler{
		db: mockDB,
	}
	ctx := context.Background()

	before := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)

	mockDB.EXPECT().AgentReserveSweep(ctx, before).Return(3, nil)

	count, err := h.ReserveSweep(ctx, before)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if count != 3 {
		t.Errorf("Wrong match. expect: 3, got: %d", count)
	}
}
