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

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_createGroupdial(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		destination  *commonaddress.Address
		callIDs      []uuid.UUID
		ringMethod   groupdial.RingMethod
		answerMethod groupdial.AnswerMethod

		responseUUID    uuid.UUID
		expectGroupdial *groupdial.Groupdial
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
			ringMethod:   groupdial.RingMethodRingAll,
			answerMethod: groupdial.AnswerMethodHangupOthers,

			responseUUID: uuid.FromStringOrNil("f57b4d16-b856-11ed-9136-27f0e7fd3764"),
			expectGroupdial: &groupdial.Groupdial{
				ID:         uuid.FromStringOrNil("f57b4d16-b856-11ed-9136-27f0e7fd3764"),
				CustomerID: uuid.FromStringOrNil("f509a864-b856-11ed-bb8e-af34cf391b5c"),
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test-exten@test-domain",
				},
				RingMethod:   groupdial.RingMethodRingAll,
				AnswerMethod: groupdial.AnswerMethodHangupOthers,
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
			mockDB.EXPECT().GroupdialCreate(ctx, tt.expectGroupdial).Return(nil)
			mockDB.EXPECT().GroupdialGet(ctx, tt.expectGroupdial.ID).Return(tt.expectGroupdial, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupdial.EventTypeGroupdialCreated, tt.expectGroupdial)

			res, err := h.createGroupdial(ctx, tt.customerID, tt.destination, tt.callIDs, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectGroupdial) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupdial, res)
			}
		})
	}
}

func Test_answerGroupdial(t *testing.T) {

	tests := []struct {
		name string

		groupdialID  uuid.UUID
		answercallID uuid.UUID

		responseGroupDial *groupdial.Groupdial
	}{
		{
			name: "normal",

			groupdialID:  uuid.FromStringOrNil("d3391861-292d-4ed8-b03a-7b455e57b17b"),
			answercallID: uuid.FromStringOrNil("1f142f05-c169-4caa-a6b2-42d224ec6ca5"),

			responseGroupDial: &groupdial.Groupdial{
				ID:           uuid.FromStringOrNil("d3391861-292d-4ed8-b03a-7b455e57b17b"),
				AnswerMethod: groupdial.AnswerMethodHangupOthers,
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

			mockDB.EXPECT().GroupdialGet(ctx, tt.groupdialID).Return(tt.responseGroupDial, nil)
			mockDB.EXPECT().GroupdialGet(ctx, tt.groupdialID).Return(tt.responseGroupDial, nil)
			updateGroupDial := *tt.responseGroupDial
			updateGroupDial.AnswerCallID = tt.answercallID
			mockDB.EXPECT().GroupdialUpdate(ctx, &updateGroupDial).Return(nil)
			mockDB.EXPECT().GroupdialGet(ctx, tt.groupdialID).Return(&updateGroupDial, nil)

			for _, callID := range tt.responseGroupDial.CallIDs {

				if callID == tt.answercallID {
					continue
				}

				// HangingUp. just return the error cause it's too long write the test code here.
				mockDB.EXPECT().CallGet(ctx, callID).Return(nil, fmt.Errorf(""))
			}

			if errAnswer := h.answerGroupdial(ctx, tt.groupdialID, tt.answercallID); errAnswer != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errAnswer)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
