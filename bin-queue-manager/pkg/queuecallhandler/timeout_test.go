package queuecallhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func Test_TimeoutWait(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("c4d753c4-ad59-11ed-ab8b-7f97d8c89352"),

			&queuecall.Queuecall{
				ID:     uuid.FromStringOrNil("c4d753c4-ad59-11ed-ab8b-7f97d8c89352"),
				Status: queuecall.StatusWaiting,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)

			// Kick()
			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(nil, fmt.Errorf(""))

			h.TimeoutWait(ctx, tt.queuecallID)
		})
	}
}

func Test_TimeoutService(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("52f489c4-ad5a-11ed-b0e7-53ea18cc4a48"),

			&queuecall.Queuecall{
				ID:     uuid.FromStringOrNil("52f489c4-ad5a-11ed-b0e7-53ea18cc4a48"),
				Status: queuecall.StatusService,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(tt.responseQueuecall, nil)

			// Kick()
			mockDB.EXPECT().QueuecallGet(ctx, tt.queuecallID).Return(nil, fmt.Errorf(""))

			h.TimeoutService(ctx, tt.queuecallID)
		})
	}
}
