package callhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func TestChainedCallIDAdd(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		id           uuid.UUID
		chaindCallID uuid.UUID
	}{
		{
			"call status progressing",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eb71954c-2504-11eb-a92f-0bd8129658a9"),
				},
				Status: call.StatusProgressing,
			},

			uuid.FromStringOrNil("eb71954c-2504-11eb-a92f-0bd8129658a9"),
			uuid.FromStringOrNil("ed893c22-2504-11eb-a0ed-839c010855ed"),
		},
		{
			"call status dialing",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a31ccbbe-256c-11eb-8d6a-7b6b14b71912"),
				},
				Status: call.StatusDialing,
			},

			uuid.FromStringOrNil("a31ccbbe-256c-11eb-8d6a-7b6b14b71912"),
			uuid.FromStringOrNil("a396a86c-256c-11eb-b3ab-d708913b7832"),
		},
		{
			"call status ringing",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a3cc1dd0-256c-11eb-afd6-871abdb9c625"),
				},
				Status: call.StatusRinging,
			},

			uuid.FromStringOrNil("a3cc1dd0-256c-11eb-afd6-871abdb9c625"),
			uuid.FromStringOrNil("a4025af8-256c-11eb-bd53-9b3a0b24844d"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallTXStart(tt.call.ID).Return(nil, tt.call, nil)
			mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.call.ID, tt.chaindCallID).Return(nil)
			mockDB.EXPECT().CallSetMasterCallID(ctx, tt.chaindCallID, tt.call.ID).Return(nil)
			mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)

			mockDB.EXPECT().CallGet(ctx, tt.chaindCallID).Return(&call.Call{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

			res, err := h.ChainedCallIDAdd(ctx, tt.id, tt.chaindCallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.call) {
				t.Errorf("Wrong match.\nexpect: %v, got: %v", tt.call, res)
			}

		})
	}
}

func TestChainedCallIDAddFailStatus(t *testing.T) {

	tests := []struct {
		name         string
		call         *call.Call
		chaindCallID uuid.UUID
	}{
		{
			"call status terminating",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa6e29e8-256d-11eb-8433-6ffc4a89784b"),
				},
				Status: call.StatusTerminating,
			},

			uuid.FromStringOrNil("aaa92570-256d-11eb-b535-8fa2fdbf4dd0"),
		},
		{
			"call status canceling",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aadeebd8-256d-11eb-8b7d-13422cdc50af"),
				},
				Status: call.StatusCanceling,
			},

			uuid.FromStringOrNil("ab13f30a-256d-11eb-bb85-e76164670011"),
		},
		{
			"call status hangup",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab4f22e0-256d-11eb-a3e1-9be571423d11"),
				},
				Status: call.StatusHangup,
			},

			uuid.FromStringOrNil("ab8784d2-256d-11eb-89fe-8f088d13b59f"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallTXStart(tt.call.ID).Return(nil, tt.call, nil)
			mockDB.EXPECT().CallTXFinish(gomock.Any(), false)

			_, err := h.ChainedCallIDAdd(ctx, tt.call.ID, tt.chaindCallID)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func TestChainedCallIDRemove(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		id           uuid.UUID
		chaindCallID uuid.UUID
	}{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4786ca48-256c-11eb-be3c-7361101fde14"),
				},
			},

			uuid.FromStringOrNil("4786ca48-256c-11eb-be3c-7361101fde14"),
			uuid.FromStringOrNil("47cb9c0e-256c-11eb-aff4-7b1173c946b1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallTXStart(tt.call.ID).Return(nil, tt.call, nil)
			mockDB.EXPECT().CallTXRemoveChainedCallID(gomock.Any(), tt.call.ID, tt.chaindCallID).Return(nil)
			mockDB.EXPECT().CallSetMasterCallID(ctx, tt.chaindCallID, uuid.Nil).Return(nil)
			mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)

			mockDB.EXPECT().CallGet(ctx, tt.chaindCallID).Return(&call.Call{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

			res, err := h.ChainedCallIDRemove(ctx, tt.call.ID, tt.chaindCallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.call, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.call, res)
			}

		})
	}
}
