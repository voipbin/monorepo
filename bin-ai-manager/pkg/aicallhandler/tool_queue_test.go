package aicallhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

// Test_toolHandleListQueues covers VOIP-1540: success with N queues,
// success with zero queues (not an error), and a downstream RPC failure.
func Test_toolHandleListQueues(t *testing.T) {

	customerID := uuid.FromStringOrNil("11111111-c001-11f0-9000-000000000001")

	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseQueues []qmqueue.Queue
		responseErr    error

		expectRes *messageContent
	}{
		{
			name: "success with queues",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-c001-11f0-9000-000000000002"),
					CustomerID: customerID,
				},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "11111111-c001-11f0-9000-000000000003",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameListQueues,
					Arguments: `{}`,
				},
			},
			responseQueues: []qmqueue.Queue{
				{
					Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-c001-11f0-9000-000000000004")},
					Name:     "sales",
					Detail:   "sales queue",
				},
			},
			expectRes: &messageContent{
				ToolCallID:   "11111111-c001-11f0-9000-000000000003",
				Result:       "success",
				ResourceType: "queue",
				Message:      `[{"id":"11111111-c001-11f0-9000-000000000004","name":"sales","detail":"sales queue"}]`,
			},
		},
		{
			name: "success with zero queues -- not an error",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-c001-11f0-9000-000000000005"),
					CustomerID: customerID,
				},
			},
			tool: &message.ToolCall{
				ID:   "11111111-c001-11f0-9000-000000000006",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameListQueues,
					Arguments: `{}`,
				},
			},
			responseQueues: []qmqueue.Queue{},
			expectRes: &messageContent{
				ToolCallID:   "11111111-c001-11f0-9000-000000000006",
				Result:       "success",
				ResourceType: "queue",
				Message:      `[]`,
			},
		},
		{
			name: "downstream RPC failure",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-c001-11f0-9000-000000000007"),
					CustomerID: customerID,
				},
			},
			tool: &message.ToolCall{
				ID:   "11111111-c001-11f0-9000-000000000008",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameListQueues,
					Arguments: `{}`,
				},
			},
			responseErr: errTest,
			expectRes: &messageContent{
				ToolCallID: "11111111-c001-11f0-9000-000000000008",
				Result:     "failed",
				Message:    errTest.Error(),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1QueueList(ctx, "", uint64(100), map[qmqueue.Field]any{
				qmqueue.FieldCustomerID: tt.aicall.CustomerID,
				qmqueue.FieldDeleted:    false,
			}).Return(tt.responseQueues, tt.responseErr)

			res := h.toolHandleListQueues(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_toolHandleJoinQueue covers VOIP-1540: happy path, wrong reference
// type, missing/invalid queue_id, queue not found, queue owned by a
// different customer (byte-identical error to not-found, IDOR masking),
// and FlowV1ActiveflowAddActions failure.
func Test_toolHandleJoinQueue(t *testing.T) {

	customerID := uuid.FromStringOrNil("22222222-c001-11f0-9000-000000000001")
	activeflowID := uuid.FromStringOrNil("22222222-c001-11f0-9000-000000000002")
	queueID := uuid.FromStringOrNil("22222222-c001-11f0-9000-000000000003")

	baseAIcall := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("22222222-c001-11f0-9000-000000000004"),
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		ReferenceType: aicall.ReferenceTypeCall,
	}

	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		skipQueueGet   bool
		responseQueue  *qmqueue.Queue
		responseGetErr error

		skipAddActions   bool
		responseAf       *fmactiveflow.Activeflow
		responseAddErr   error

		expectRes *messageContent
	}{
		{
			name:   "happy path",
			aicall: baseAIcall,
			tool: &message.ToolCall{
				ID:   "22222222-c001-11f0-9000-000000000005",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameJoinQueue,
					Arguments: `{"queue_id":"` + queueID.String() + `"}`,
				},
			},
			responseQueue: &qmqueue.Queue{
				Identity: commonidentity.Identity{ID: queueID, CustomerID: customerID},
			},
			responseAf: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{ID: activeflowID},
			},
			expectRes: &messageContent{
				ToolCallID:   "22222222-c001-11f0-9000-000000000005",
				Result:       "success",
				ResourceType: "activeflow",
				ResourceID:   activeflowID.String(),
				Message:      "Added queue_join action successfully.",
			},
		},
		{
			name: "wrong reference type",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: baseAIcall.ID, CustomerID: customerID},
				ActiveflowID:  activeflowID,
				ReferenceType: aicall.ReferenceTypeConversation,
			},
			tool: &message.ToolCall{
				ID:   "22222222-c001-11f0-9000-000000000006",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameJoinQueue,
					Arguments: `{"queue_id":"` + queueID.String() + `"}`,
				},
			},
			skipQueueGet:   true,
			skipAddActions: true,
			expectRes: &messageContent{
				ToolCallID: "22222222-c001-11f0-9000-000000000006",
				Result:     "failed",
				Message:    "join_queue is only supported for call reference type",
			},
		},
		{
			name:   "missing queue_id",
			aicall: baseAIcall,
			tool: &message.ToolCall{
				ID:   "22222222-c001-11f0-9000-000000000007",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameJoinQueue,
					Arguments: `{}`,
				},
			},
			skipQueueGet:   true,
			skipAddActions: true,
			expectRes: &messageContent{
				ToolCallID: "22222222-c001-11f0-9000-000000000007",
				Result:     "failed",
				Message:    "queue_id is required",
			},
		},
		{
			name:   "queue not found",
			aicall: baseAIcall,
			tool: &message.ToolCall{
				ID:   "22222222-c001-11f0-9000-000000000008",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameJoinQueue,
					Arguments: `{"queue_id":"` + queueID.String() + `"}`,
				},
			},
			responseGetErr: errTest,
			skipAddActions: true,
			expectRes: &messageContent{
				ToolCallID: "22222222-c001-11f0-9000-000000000008",
				Result:     "failed",
				Message:    errQueueNotResolvable.Error(),
			},
		},
		{
			name:   "queue owned by a different customer -- byte-identical to not-found",
			aicall: baseAIcall,
			tool: &message.ToolCall{
				ID:   "22222222-c001-11f0-9000-000000000009",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameJoinQueue,
					Arguments: `{"queue_id":"` + queueID.String() + `"}`,
				},
			},
			responseQueue: &qmqueue.Queue{
				Identity: commonidentity.Identity{
					ID:         queueID,
					CustomerID: uuid.FromStringOrNil("22222222-c001-11f0-9000-0000000000ff"), // different customer
				},
			},
			skipAddActions: true,
			expectRes: &messageContent{
				ToolCallID: "22222222-c001-11f0-9000-000000000009",
				Result:     "failed",
				Message:    errQueueNotResolvable.Error(),
			},
		},
		{
			name:   "FlowV1ActiveflowAddActions error",
			aicall: baseAIcall,
			tool: &message.ToolCall{
				ID:   "22222222-c001-11f0-9000-00000000000a",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameJoinQueue,
					Arguments: `{"queue_id":"` + queueID.String() + `"}`,
				},
			},
			responseQueue: &qmqueue.Queue{
				Identity: commonidentity.Identity{ID: queueID, CustomerID: customerID},
			},
			responseAddErr: errTest,
			expectRes: &messageContent{
				ToolCallID: "22222222-c001-11f0-9000-00000000000a",
				Result:     "failed",
				Message:    errTest.Error(),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			if !tt.skipQueueGet {
				mockReq.EXPECT().QueueV1QueueGet(ctx, queueID).Return(tt.responseQueue, tt.responseGetErr)
			}

			if !tt.skipAddActions && tt.responseGetErr == nil && tt.responseQueue != nil && tt.responseQueue.CustomerID == tt.aicall.CustomerID {
				opt := fmaction.OptionQueueJoin{QueueID: queueID}
				actions := []fmaction.Action{
					{Type: fmaction.TypeQueueJoin, Option: fmaction.ConvertOption(opt)},
				}
				mockReq.EXPECT().FlowV1ActiveflowAddActions(ctx, tt.aicall.ActiveflowID, actions).Return(tt.responseAf, tt.responseAddErr)
				if tt.responseAddErr == nil {
					// join_queue fires a fire-and-forget goroutine to terminate the
					// aicall; allow (but do not require) the call in this unit test.
					mockReq.EXPECT().AIV1AIcallTerminate(gomock.Any(), tt.aicall.ID).Return(nil, nil).AnyTimes()
				}
			}

			res := h.toolHandleJoinQueue(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
