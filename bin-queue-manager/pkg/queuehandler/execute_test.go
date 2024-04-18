package queuehandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

func Test_Execute(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID

		responseQueue     *queue.Queue
		responseCurTime   string
		responseQueuecall []queuecall.Queuecall
		responseAgent     []amagent.Agent

		expectFiltersQueue map[string]string
		expectFiltersAgent map[string]string
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

			map[string]string{
				"queue_id": "558dc9da-d1ae-11ec-b9f8-e323caeb57c4",
				"status":   string(queuecall.StatusWaiting),
			},
			map[string]string{
				"deleted":     "false",
				"customer_id": "a3361ad8-d1af-11ec-865d-cf7070170a25",
				"tag_ids":     "a3a6841c-d1af-11ec-8844-c7602a790709",
				"status":      string(amagent.StatusAvailable),
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
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockReq.EXPECT().QueueV1QueuecallGets(ctx, tt.responseCurTime, uint64(1), tt.expectFiltersQueue).Return(tt.responseQueuecall, nil)

			// GetAgents
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockReq.EXPECT().AgentV1AgentGets(ctx, gomock.Any(), uint64(100), tt.expectFiltersAgent).Return(tt.responseAgent, nil)

			mockReq.EXPECT().QueueV1QueuecallExecute(ctx, tt.responseQueuecall[0].ID, gomock.Any()).Return(&queuecall.Queuecall{}, nil)
			mockReq.EXPECT().QueueV1QueueExecuteRun(ctx, tt.queueID, 100)

			h.Execute(ctx, tt.queueID)
		})
	}
}
