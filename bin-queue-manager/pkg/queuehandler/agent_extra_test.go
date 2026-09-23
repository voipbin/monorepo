package queuehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

func Test_hasCommonTagID(t *testing.T) {
	tagA := uuid.FromStringOrNil("5d443cfe-b499-11ec-ac74-83f95d8a0381")
	tagB := uuid.FromStringOrNil("4fc21d6c-b244-11ee-9bd1-1b47f77edd77")
	tagC := uuid.FromStringOrNil("60d3d9de-0000-11ee-0000-000000000001")

	tests := []struct {
		name string

		a []uuid.UUID
		b []uuid.UUID

		expectRes bool
	}{
		{
			"overlap -- one shared id (OR semantics, not all)",
			[]uuid.UUID{tagA, tagB},
			[]uuid.UUID{tagB, tagC},
			true,
		},
		{
			"no overlap",
			[]uuid.UUID{tagA},
			[]uuid.UUID{tagC},
			false,
		},
		{
			"both empty",
			[]uuid.UUID{},
			[]uuid.UUID{},
			false,
		},
		{
			"a empty",
			[]uuid.UUID{},
			[]uuid.UUID{tagA},
			false,
		},
		{
			"b empty",
			[]uuid.UUID{tagA},
			[]uuid.UUID{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := hasCommonTagID(tt.a, tt.b)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GetQueuesByAgent(t *testing.T) {
	tagA := uuid.FromStringOrNil("5d443cfe-b499-11ec-ac74-83f95d8a0381")
	tagB := uuid.FromStringOrNil("4fc21d6c-b244-11ee-9bd1-1b47f77edd77")
	customerID := uuid.FromStringOrNil("dd185d70-b499-11ec-a4b6-735983739876")

	tests := []struct {
		name string

		agent amagent.Agent

		responseQueues []*queue.Queue

		expectFilters map[queue.Field]any
		expectRes     []*queue.Queue
	}{
		{
			"single page -- untagged queue always included, tagged queue matched by overlap, tagged queue with no overlap excluded",

			amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("624e1cd6-d1b0-11ec-8b3b-db12aa2e35f6"),
					CustomerID: customerID,
				},
				TagIDs: []uuid.UUID{tagA},
			},

			[]*queue.Queue{
				{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("10b6bd90-b49d-11ec-950c-d3213b7e8cda")},
					TagIDs:   []uuid.UUID{},
				},
				{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("20b6bd90-b49d-11ec-950c-d3213b7e8cda")},
					TagIDs:   []uuid.UUID{tagA},
				},
				{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("30b6bd90-b49d-11ec-950c-d3213b7e8cda")},
					TagIDs:   []uuid.UUID{tagB},
				},
			},

			map[queue.Field]any{
				queue.FieldDeleted:    false,
				queue.FieldCustomerID: customerID.String(),
			},
			[]*queue.Queue{
				{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("10b6bd90-b49d-11ec-950c-d3213b7e8cda")},
					TagIDs:   []uuid.UUID{},
				},
				{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("20b6bd90-b49d-11ec-950c-d3213b7e8cda")},
					TagIDs:   []uuid.UUID{tagA},
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

			mockDB.EXPECT().QueueList(ctx, uint64(queueListPageSize), "", tt.expectFilters).Return(tt.responseQueues, nil)

			res, err := h.GetQueuesByAgent(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_GetQueuesByAgent_pagination verifies the page walk continues past a
// full page (len(qs) == queueListPageSize) and stops on a short page,
// carrying the token forward as the last row's tm_create (VOIP-1539 §3.5).
func Test_GetQueuesByAgent_pagination(t *testing.T) {
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
	agent := amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("624e1cd6-d1b0-11ec-8b3b-db12aa2e35f6"),
			CustomerID: uuid.FromStringOrNil("dd185d70-b499-11ec-a4b6-735983739876"),
		},
	}

	filters := map[queue.Field]any{
		queue.FieldDeleted:    false,
		queue.FieldCustomerID: agent.CustomerID.String(),
	}

	firstPageTM := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	fullPage := make([]*queue.Queue, queueListPageSize)
	for i := range fullPage {
		fullPage[i] = &queue.Queue{
			Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
			TagIDs:   []uuid.UUID{},
			TMCreate: &firstPageTM,
		}
	}
	secondPage := []*queue.Queue{
		{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("30b6bd90-b49d-11ec-950c-d3213b7e8cda")},
			TagIDs:   []uuid.UUID{},
		},
	}

	gomock.InOrder(
		mockDB.EXPECT().QueueList(ctx, uint64(queueListPageSize), "", filters).Return(fullPage, nil),
		mockDB.EXPECT().QueueList(ctx, uint64(queueListPageSize), firstPageTM.UTC().Format(utilhandler.ISO8601Layout), filters).Return(secondPage, nil),
	)

	res, err := h.GetQueuesByAgent(ctx, agent)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != queueListPageSize+1 {
		t.Errorf("Wrong match. expect: %d, got: %d", queueListPageSize+1, len(res))
	}
}
