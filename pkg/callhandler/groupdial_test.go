package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_createGroupcall(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		destination  *commonaddress.Address
		callIDs      []uuid.UUID
		ringMethod   groupcall.RingMethod
		answerMethod groupcall.AnswerMethod

		responseUUID    uuid.UUID
		expectGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("f509a864-b856-11ed-bb8e-af34cf391b5c"),
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeEndpoint,
				Target: "test-exten@test-domain",
			},
			callIDs: []uuid.UUID{
				uuid.FromStringOrNil("f530680a-b856-11ed-ab4d-5f86afc4f7c1"),
				uuid.FromStringOrNil("f5551592-b856-11ed-9902-2fbfea849d8a"),
			},
			ringMethod:   groupcall.RingMethodRingAll,
			answerMethod: groupcall.AnswerMethodHangupOthers,

			responseUUID: uuid.FromStringOrNil("f57b4d16-b856-11ed-9136-27f0e7fd3764"),
			expectGroupcall: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("f57b4d16-b856-11ed-9136-27f0e7fd3764"),
				CustomerID: uuid.FromStringOrNil("f509a864-b856-11ed-bb8e-af34cf391b5c"),
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test-exten@test-domain",
				},
				RingMethod:   groupcall.RingMethodRingAll,
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f530680a-b856-11ed-ab4d-5f86afc4f7c1"),
					uuid.FromStringOrNil("f5551592-b856-11ed-9902-2fbfea849d8a"),
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().GroupcallCreate(ctx, tt.expectGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.expectGroupcall.ID).Return(tt.expectGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallCreated, tt.expectGroupcall)

			res, err := h.createGroupcall(ctx, tt.customerID, tt.destination, tt.callIDs, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupcall, res)
			}
		})
	}
}

func Test_answerGroupcall(t *testing.T) {

	tests := []struct {
		name string

		groupcallID  uuid.UUID
		answercallID uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			groupcallID:  uuid.FromStringOrNil("d3391861-292d-4ed8-b03a-7b455e57b17b"),
			answercallID: uuid.FromStringOrNil("1f142f05-c169-4caa-a6b2-42d224ec6ca5"),

			responseGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("d3391861-292d-4ed8-b03a-7b455e57b17b"),
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f3a4b38b-4781-4db7-b74a-5958b2851225"),
					uuid.FromStringOrNil("0de4689f-96d5-448d-be00-11c196163756"),
					uuid.FromStringOrNil("1f142f05-c169-4caa-a6b2-42d224ec6ca5"),
				},
			},
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

			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			updateGroupcall := *tt.responseGroupcall
			updateGroupcall.AnswerCallID = tt.answercallID
			mockDB.EXPECT().GroupcallUpdate(ctx, &updateGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(&updateGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallAnswered, &updateGroupcall)

			for _, callID := range tt.responseGroupcall.CallIDs {
				if callID == tt.answercallID {
					continue
				}

				// HangingUp()
				mockDB.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf(""))
			}

			if errAnswer := h.answerGroupcall(ctx, tt.groupcallID, tt.answercallID); errAnswer != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errAnswer)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_updateGroupcallAnswerCallID(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		callID uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("87f62caa-a188-457a-be85-41147833a012"),
			callID: uuid.FromStringOrNil("f864a3df-166e-4cfa-8db1-64ad035d7d89"),

			responseGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("87f62caa-a188-457a-be85-41147833a012"),
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("2fb56182-75a4-4868-9e3c-09ffab0ed7dc"),
					uuid.FromStringOrNil("29cf22c0-51cb-4eb1-a684-7dcb083b3a81"),
					uuid.FromStringOrNil("f864a3df-166e-4cfa-8db1-64ad035d7d89"),
				},
			},
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

			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)
			updateGroupcall := *tt.responseGroupcall
			updateGroupcall.AnswerCallID = tt.callID
			mockDB.EXPECT().GroupcallUpdate(ctx, &updateGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(&updateGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallAnswered, &updateGroupcall)

			res, err := h.updateGroupcallAnswerCallID(ctx, tt.id, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, &updateGroupcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updateGroupcall, res)
			}
		})
	}
}
