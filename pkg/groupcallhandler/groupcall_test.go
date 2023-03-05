package groupcallhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		source       *commonaddress.Address
		destinations []commonaddress.Address
		callIDs      []uuid.UUID
		ringMethod   groupcall.RingMethod
		answerMethod groupcall.AnswerMethod

		responseUUID    uuid.UUID
		expectGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("c345ddd8-bb27-11ed-812c-df4f74c7c1a1"),
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			callIDs: []uuid.UUID{
				uuid.FromStringOrNil("c38fe31a-bb27-11ed-9e5c-bf52e856e97c"),
				uuid.FromStringOrNil("c3c0630a-bb27-11ed-a026-236fa4f96287"),
			},
			ringMethod:   groupcall.RingMethodRingAll,
			answerMethod: groupcall.AnswerMethodHangupOthers,

			responseUUID: uuid.FromStringOrNil("c3f3d65e-bb27-11ed-8e79-d74dd86a55ba"),
			expectGroupcall: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("c3f3d65e-bb27-11ed-8e79-d74dd86a55ba"),
				CustomerID: uuid.FromStringOrNil("c345ddd8-bb27-11ed-812c-df4f74c7c1a1"),
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				RingMethod:   groupcall.RingMethodRingAll,
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				AnswerCallID: [16]byte{},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("c38fe31a-bb27-11ed-9e5c-bf52e856e97c"),
					uuid.FromStringOrNil("c3c0630a-bb27-11ed-a026-236fa4f96287"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().GroupcallCreate(ctx, tt.expectGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.expectGroupcall.ID).Return(tt.expectGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallCreated, tt.expectGroupcall)

			res, err := h.Create(ctx, tt.customerID, tt.source, tt.destinations, tt.callIDs, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupcall, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("678717bc-bb29-11ed-81c0-c3d2e4da7296"),
			responseGroupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("678717bc-bb29-11ed-81c0-c3d2e4da7296"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_UpdateAnswerCallID(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		callID uuid.UUID

		responseGroupcall *groupcall.Groupcall
		expectGroupcall   *groupcall.Groupcall
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("ac6ae0fc-bb29-11ed-9f2a-6b95feacf142"),
			callID: uuid.FromStringOrNil("ac9c3c42-bb29-11ed-aa47-47441a47de62"),

			responseGroupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("ac6ae0fc-bb29-11ed-9f2a-6b95feacf142"),
			},
			expectGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("ac6ae0fc-bb29-11ed-9f2a-6b95feacf142"),
				AnswerCallID: uuid.FromStringOrNil("ac9c3c42-bb29-11ed-aa47-47441a47de62"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			mockDB.EXPECT().GroupcallUpdate(ctx, tt.expectGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.expectGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallAnswered, tt.expectGroupcall)

			res, err := h.UpdateAnswerCallID(ctx, tt.id, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}
