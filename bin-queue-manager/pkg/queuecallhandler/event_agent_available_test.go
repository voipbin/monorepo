package queuecallhandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

// Test_EventAMAgentAvailable_NoEligibleQueues verifies the entry-point-B
// no-op path (VOIP-1539 §3.5): when the agent has no eligible queues, no
// queuecall lookup or match attempt is made.
func Test_EventAMAgentAvailable_NoEligibleQueues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &queuecallHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyhandler: mockNotify,
		queueHandler:  mockQueue,
	}

	ctx := context.Background()
	agent := amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("624e1cd6-d1b0-11ec-8b3b-db12aa2e35f6")},
		Status:   amagent.StatusAvailable,
	}

	mockQueue.EXPECT().GetQueuesByAgent(ctx, agent).Return([]*queue.Queue{}, nil)

	// No further mock expectations: QueuecallListOldestWaiting must not be called.
	h.EventAMAgentAvailable(ctx, agent)
}

// Test_EventAMAgentAvailable_NoWaitingQueuecall verifies that an eligible
// queue with zero waiting queuecalls is skipped without attempting a match.
func Test_EventAMAgentAvailable_NoWaitingQueuecall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &queuecallHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyhandler: mockNotify,
		queueHandler:  mockQueue,
	}

	ctx := context.Background()
	agent := amagent.Agent{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("624e1cd6-d1b0-11ec-8b3b-db12aa2e35f6")},
		Status:   amagent.StatusAvailable,
	}
	q := &queue.Queue{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("10b6bd90-b49d-11ec-950c-d3213b7e8cda")}}

	mockQueue.EXPECT().GetQueuesByAgent(ctx, agent).Return([]*queue.Queue{q}, nil)
	mockDB.EXPECT().QueuecallListOldestWaiting(ctx, q.ID, uint64(1)).Return([]*queuecall.Queuecall{}, nil)

	// No Execute()-path mocks: since qcs is empty, Execute must not be called
	// (would panic on unexpected mock calls if it were).
	h.EventAMAgentAvailable(ctx, agent)
}
