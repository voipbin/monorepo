package queuecallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

func Test_ServiceStart(t *testing.T) {

	tests := []struct {
		name string

		queueID       uuid.UUID
		activeflowID  uuid.UUID
		referenceType queuecall.ReferenceType
		referenceID   uuid.UUID

		responseQueue         *queue.Queue
		responseCall          *cmcall.Call
		responseConfbridge    *cmconfbridge.Confbridge
		responseUUIDTargetID  uuid.UUID
		responseUUIDLoopID    uuid.UUID
		responseUUIDForwardID uuid.UUID
		responseUUIDQueuecall uuid.UUID
		responseQueuecall     *queuecall.Queuecall

		expectQueuecall *queuecall.Queuecall
		expectRes       *commonservice.Service
	}{
		{
			name: "normal",

			queueID:       uuid.FromStringOrNil("e7d1c428-acef-11ed-9009-f32fafb30091"),
			activeflowID:  uuid.FromStringOrNil("e8004cda-acef-11ed-8af6-1f155a5daa45"),
			referenceType: queuecall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e82487ee-acef-11ed-b6a0-d375ffdc940c"),

			responseQueue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e7d1c428-acef-11ed-9009-f32fafb30091"),
					CustomerID: uuid.FromStringOrNil("525c62ba-acf2-11ed-a514-c7d0b90804bb"),
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs:        []uuid.UUID{},
				WaitFlowID:    uuid.FromStringOrNil("eef35a2a-20c2-11f0-8352-4b8bc8a17a09"),
			},
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e82487ee-acef-11ed-b6a0-d375ffdc940c"),
				},
			},
			responseUUIDQueuecall: uuid.FromStringOrNil("b0fd8d1e-acf0-11ed-9430-6f880d5c9104"),
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0c77a26-acf0-11ed-8fd7-37de63b3d029"),
				},
			},
			responseUUIDTargetID:  uuid.FromStringOrNil("239d5d9e-acf2-11ed-96d1-8b6af7ef84bd"),
			responseUUIDLoopID:    uuid.FromStringOrNil("8dc3047c-20c6-11f0-9d4a-8787d9e080b9"),
			responseUUIDForwardID: uuid.FromStringOrNil("8e22e766-20c6-11f0-8e98-2b6dd89c825c"),
			responseQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0fd8d1e-acf0-11ed-9430-6f880d5c9104"),
				},
			},

			expectQueuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0fd8d1e-acf0-11ed-9430-6f880d5c9104"),
					CustomerID: uuid.FromStringOrNil("525c62ba-acf2-11ed-a514-c7d0b90804bb"),
				},
				QueueID:               uuid.FromStringOrNil("e7d1c428-acef-11ed-9009-f32fafb30091"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("e82487ee-acef-11ed-b6a0-d375ffdc940c"),
				ReferenceActiveflowID: uuid.FromStringOrNil("e8004cda-acef-11ed-8af6-1f155a5daa45"),
				ForwardActionID:       uuid.FromStringOrNil("8e22e766-20c6-11f0-8e98-2b6dd89c825c"),
				ConfbridgeID:          uuid.FromStringOrNil("b0c77a26-acf0-11ed-8fd7-37de63b3d029"),
				Source:                commonaddress.Address{},
				RoutingMethod:         queue.RoutingMethodRandom,
				TagIDs:                []uuid.UUID{},
				Status:                queuecall.StatusInitiating,
				ServiceAgentID:        uuid.Nil,
				TimeoutWait:           0,
				TimeoutService:        0,
				DurationWaiting:       0,
				DurationService:       0,
			},
			expectRes: &commonservice.Service{
				ID:   uuid.FromStringOrNil("b0fd8d1e-acf0-11ed-9430-6f880d5c9104"),
				Type: commonservice.TypeQueuecall,
				PushActions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("239d5d9e-acf2-11ed-96d1-8b6af7ef84bd"),
						Type: fmaction.TypeFetchFlow,
						Option: fmaction.ConvertOption(fmaction.OptionFetchFlow{
							FlowID: uuid.FromStringOrNil("eef35a2a-20c2-11f0-8352-4b8bc8a17a09"),
						}),
					},
					{
						ID:     uuid.FromStringOrNil("8dc3047c-20c6-11f0-9d4a-8787d9e080b9"),
						Type:   fmaction.TypeEmpty,
						Option: fmaction.ConvertOption(fmaction.OptionEmpty{}),
						NextID: uuid.FromStringOrNil("239d5d9e-acf2-11ed-96d1-8b6af7ef84bd"),
					},
					{
						ID:   uuid.FromStringOrNil("8e22e766-20c6-11f0-8e98-2b6dd89c825c"),
						Type: fmaction.TypeConfbridgeJoin,
						Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
							ConfbridgeID: uuid.FromStringOrNil("b0c77a26-acf0-11ed-8fd7-37de63b3d029"),
						}),
					},
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
			mockQueue := queuehandler.NewMockQueueHandler(mc)

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,

				queueHandler: mockQueue,
			}

			ctx := context.Background()

			mockQueue.EXPECT().Get(ctx, tt.queueID.Return(tt.responseQueue, nil)
			mockReq.EXPECT().CallV1CallGet(ctx, tt.referenceID.Return(tt.responseCall, nil)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDQueuecall)
			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.responseQueue.CustomerID, tt.activeflowID, cmconfbridge.ReferenceTypeQueue, tt.responseUUIDQueuecall, cmconfbridge.TypeConnect.Return(tt.responseConfbridge, nil)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDTargetID)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDLoopID)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDForwardID)
			mockDB.EXPECT().QueuecallCreate(ctx, tt.expectQueuecall.Return(nil)
			mockDB.EXPECT().QueuecallGet(ctx, tt.responseUUIDQueuecall.Return(tt.responseQueuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseQueuecall.CustomerID, queuecall.EventTypeQueuecallCreated, tt.responseQueuecall)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.responseQueuecall.ReferenceActiveflowID, gomock.Any().Return(nil)
			mockReq.EXPECT().QueueV1QueuecallHealthCheck(ctx, tt.responseQueuecall.ID, defaultHealthCheckDelay, 0.Return(nil)

			res, err := h.ServiceStart(ctx, tt.queueID, tt.activeflowID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_createActions(t *testing.T) {

	tests := []struct {
		name string

		queue        *queue.Queue
		confbridgeID uuid.UUID

		responseUUIDTargetID  uuid.UUID
		responseUUIDLoopID    uuid.UUID
		responseUUIDForwardID uuid.UUID

		expectRes []fmaction.Action
	}{
		{
			name: "normal",

			queue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("61a4651c-60e3-11ec-86ff-efca21ef8707"),
				},
				WaitFlowID: uuid.FromStringOrNil("f5701776-20c2-11f0-9c76-2342e18ec235"),
			},
			confbridgeID: uuid.FromStringOrNil("9c758344-81a6-48b1-be2b-5128e2579a9c"),

			responseUUIDTargetID:  uuid.FromStringOrNil("61d32f5a-60e3-11ec-943d-db1b16329a1c"),
			responseUUIDLoopID:    uuid.FromStringOrNil("f5953d3a-20c2-11f0-9fcb-63111e54a3c4"),
			responseUUIDForwardID: uuid.FromStringOrNil("f5b7281e-20c2-11f0-afb6-736645327c60"),

			expectRes: []fmaction.Action{
				{
					ID:   uuid.FromStringOrNil("61d32f5a-60e3-11ec-943d-db1b16329a1c"),
					Type: fmaction.TypeFetchFlow,
					Option: fmaction.ConvertOption(fmaction.OptionFetchFlow{
						FlowID: uuid.FromStringOrNil("f5701776-20c2-11f0-9c76-2342e18ec235"),
					}),
				},
				{
					ID:     uuid.FromStringOrNil("f5953d3a-20c2-11f0-9fcb-63111e54a3c4"),
					Type:   fmaction.TypeEmpty,
					Option: fmaction.ConvertOption(fmaction.OptionEmpty{}),
					NextID: uuid.FromStringOrNil("61d32f5a-60e3-11ec-943d-db1b16329a1c"),
				},
				{
					ID:   uuid.FromStringOrNil("f5b7281e-20c2-11f0-afb6-736645327c60"),
					Type: fmaction.TypeConfbridgeJoin,
					Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
						ConfbridgeID: uuid.FromStringOrNil("9c758344-81a6-48b1-be2b-5128e2579a9c"),
					}),
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

			h := &queuecallHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDTargetID)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDLoopID)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDForwardID)

			res, resForward, err := h.createActions(ctx, tt.queue, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			if resForward != tt.responseUUIDForwardID {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseUUIDForwardID, resForward)
			}
		})
	}
}
