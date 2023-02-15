package queuehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func Test_Execute(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID

		responseQueue     *queue.Queue
		responseCurTime   string
		responseQueuecall []queuecall.Queuecall
		responseAgent     []amagent.Agent
	}{
		{
			"normal",

			uuid.FromStringOrNil("558dc9da-d1ae-11ec-b9f8-e323caeb57c4"),

			&queue.Queue{
				ID:         uuid.FromStringOrNil("558dc9da-d1ae-11ec-b9f8-e323caeb57c4"),
				CustomerID: uuid.FromStringOrNil("a3361ad8-d1af-11ec-865d-cf7070170a25"),
				Execute:    queue.ExecuteRun,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a3a6841c-d1af-11ec-8844-c7602a790709"),
				},
				RoutingMethod: queue.RoutingMethodRandom,
			},
			"2023-02-14 03:22:17.995000",
			[]queuecall.Queuecall{
				{
					ID: uuid.FromStringOrNil("0313ffe8-d1af-11ec-a1e7-3b1e1fb76015"),
				},
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("7c8e7e02-d1af-11ec-8d8e-d7280dd6fcc8"),
				},
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

			h := &queueHandler{
				utilhandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockReq.EXPECT().QueueV1QueuecallGetsByQueueIDAndStatus(ctx, tt.responseQueue.ID, queuecall.StatusWaiting, tt.responseCurTime, uint64(1)).Return(tt.responseQueuecall, nil)

			// GetAgents
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockReq.EXPECT().AgentV1AgentGetsByTagIDsAndStatus(ctx, tt.responseQueue.CustomerID, tt.responseQueue.TagIDs, amagent.StatusAvailable).Return(tt.responseAgent, nil)

			mockReq.EXPECT().QueueV1QueuecallExecute(ctx, tt.responseQueuecall[0].ID, gomock.Any()).Return(&queuecall.Queuecall{}, nil)
			mockReq.EXPECT().QueueV1QueueExecuteRun(ctx, tt.queueID, 100)

			h.Execute(ctx, tt.queueID)
		})
	}
}
