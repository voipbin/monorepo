package queuehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func Test_GetAgents(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status amagent.Status

		responseQueue  *queue.Queue
		responseAgents []amagent.Agent

		expectFilters map[string]string
		expectRes     []amagent.Agent
	}{
		{
			"none",

			uuid.FromStringOrNil("10b6bd90-b49d-11ec-950c-d3213b7e8cda"),
			amagent.StatusNone,

			&queue.Queue{
				ID:         uuid.FromStringOrNil("10b6bd90-b49d-11ec-950c-d3213b7e8cda"),
				CustomerID: uuid.FromStringOrNil("dd185d70-b499-11ec-a4b6-735983739876"),
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("5d443cfe-b499-11ec-ac74-83f95d8a0381"),
					uuid.FromStringOrNil("4fc21d6c-b244-11ee-9bd1-1b47f77edd77"),
				},
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("5d66c59e-b499-11ec-9109-dfdab27cf4e1"),
				},
			},

			map[string]string{
				"deleted":     "false",
				"customer_id": "dd185d70-b499-11ec-a4b6-735983739876",
				"tag_ids":     "5d443cfe-b499-11ec-ac74-83f95d8a0381,4fc21d6c-b244-11ee-9bd1-1b47f77edd77",
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("5d66c59e-b499-11ec-9109-dfdab27cf4e1"),
				},
			},
		},
		{
			"available",

			uuid.FromStringOrNil("10f079ea-b49d-11ec-9eae-9bf3cacd1f17"),
			amagent.StatusAvailable,

			&queue.Queue{
				ID:         uuid.FromStringOrNil("10f079ea-b49d-11ec-9eae-9bf3cacd1f17"),
				CustomerID: uuid.FromStringOrNil("dd185d70-b499-11ec-a4b6-735983739876"),
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("5d443cfe-b499-11ec-ac74-83f95d8a0381"),
				},
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("5d66c59e-b499-11ec-9109-dfdab27cf4e1"),
				},
			},

			map[string]string{
				"deleted":     "false",
				"customer_id": "dd185d70-b499-11ec-a4b6-735983739876",
				"status":      string(amagent.StatusAvailable),
				"tag_ids":     "5d443cfe-b499-11ec-ac74-83f95d8a0381",
			},
			[]amagent.Agent{
				{
					ID: uuid.FromStringOrNil("5d66c59e-b499-11ec-9109-dfdab27cf4e1"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			mockDB.EXPECT().QueueGet(ctx, tt.id).Return(tt.responseQueue, nil)

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockReq.EXPECT().AgentV1AgentGets(ctx, gomock.Any(), uint64(100), tt.expectFilters).Return(tt.responseAgents, nil)

			res, err := h.GetAgents(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
