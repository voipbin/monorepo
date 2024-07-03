package groupcallhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_HangingupOthers(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall

		responseGroupcalls []*groupcall.Groupcall
	}{
		{
			name: "normal",

			groupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("da99d4e8-d905-11ed-8a4c-a72c1eb8b80f"),
				AnswerCallID: uuid.FromStringOrNil("db0412c2-d905-11ed-b350-272f54423bec"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("db0412c2-d905-11ed-b350-272f54423bec"),
					uuid.FromStringOrNil("db2f3c86-d905-11ed-aa9e-d7752d9d4d3f"),
				},

				AnswerGroupcallID: uuid.FromStringOrNil("320a59da-e2bd-11ed-a55b-831465014a82"),
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("320a59da-e2bd-11ed-a55b-831465014a82"),
					uuid.FromStringOrNil("3254b16a-e2bd-11ed-b31e-df49f649152a"),
				},
			},

			responseGroupcalls: []*groupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("3254b16a-e2bd-11ed-b31e-df49f649152a"),
					CallIDs: []uuid.UUID{
						uuid.FromStringOrNil("71912f2a-e2bd-11ed-b1f8-fba9db2da3bc"),
						uuid.FromStringOrNil("71c31706-e2bd-11ed-b7bb-53bb2e2dfc2e"),
					},
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			for _, groupcallID := range tt.groupcall.GroupcallIDs {
				mockReq.EXPECT().CallV1GroupcallHangupOthers(ctx, groupcallID).Return(nil)
			}

			for _, callID := range tt.groupcall.CallIDs {
				if callID == tt.groupcall.AnswerCallID {
					continue
				}
				mockReq.EXPECT().CallV1CallHangup(ctx, callID).Return(&call.Call{}, nil)
			}

			if errHangup := h.HangingupOthers(ctx, tt.groupcall); errHangup != nil {
				t.Errorf("Wrong match.\nexpect: nil\ngot: %v", errHangup)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_Hangingup(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall  *groupcall.Groupcall
		responseGroupcalls []*groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("81c68518-e2c0-11ed-bcf0-7f41a7a2a0fd"),

			responseGroupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("81c68518-e2c0-11ed-bcf0-7f41a7a2a0fd"),

				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("688c37a4-e2c1-11ed-aa37-43c89a764ffa"),
					uuid.FromStringOrNil("68ba0328-e2c1-11ed-9b95-8f8658666e47"),
				},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("24e1c622-e2c1-11ed-9def-2bb7603769b0"),
					uuid.FromStringOrNil("250935ea-e2c1-11ed-85de-fff8ed66d29e"),
				},
			},
			responseGroupcalls: []*groupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("688c37a4-e2c1-11ed-aa37-43c89a764ffa"),
					CallIDs: []uuid.UUID{
						uuid.FromStringOrNil("b425439a-e2c1-11ed-84d5-3f9f98878242"),
						uuid.FromStringOrNil("b44c7406-e2c1-11ed-baeb-9f9c34586579"),
					},
				},
				{
					ID: uuid.FromStringOrNil("250935ea-e2c1-11ed-85de-fff8ed66d29e"),
					CallIDs: []uuid.UUID{
						uuid.FromStringOrNil("b4778434-e2c1-11ed-8968-136b2bfe7bdc"),
						uuid.FromStringOrNil("b4a2917e-e2c1-11ed-b462-cbd0377dfcce"),
					},
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.id, groupcall.StatusHangingup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			// groupcall hanging up
			for _, groupcallID := range tt.responseGroupcall.GroupcallIDs {
				mockReq.EXPECT().CallV1GroupcallHangup(ctx, groupcallID).Return(&groupcall.Groupcall{}, nil)
			}

			// calls hanging up
			for _, callID := range tt.responseGroupcall.CallIDs {
				mockReq.EXPECT().CallV1CallHangup(ctx, callID).Return(&call.Call{}, nil)
			}

			res, err := h.Hangingup(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: nil\ngot: %v", err)
			}

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_Hangup(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("c3892bf4-e270-11ed-8b11-1bfcdb4ba661"),

			responseGroupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("c3892bf4-e270-11ed-8b11-1bfcdb4ba661"),
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.id, groupcall.StatusHangup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallHangup, tt.responseGroupcall)

			res, err := h.Hangup(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: nil\ngot: %v", err)
			}

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_hangupRingMethodLinear_mastercall_has_invalid_status(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall

		responseCall *call.Call
	}{
		{
			name: "has master call info and status is canceling",

			groupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("539dbd0a-e26b-11ed-b6d7-4ff6c9abd671"),
				MasterCallID: uuid.FromStringOrNil("8b39b79c-e26a-11ed-a15d-a70fae1d453a"),
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8b39b79c-e26a-11ed-a15d-a70fae1d453a"),
				},
				Status: call.StatusCanceling,
			},
		},
		{
			name: "has master call info and status is terminating",

			groupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("75d18f64-e26b-11ed-982a-43572f221e34"),
				MasterCallID: uuid.FromStringOrNil("75f8aef0-e26b-11ed-90f9-5bdeb6afde1e"),
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("75f8aef0-e26b-11ed-90f9-5bdeb6afde1e"),
				},
				Status: call.StatusTerminating,
			},
		},
		{
			name: "has master call info and status is hangup",

			groupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("902c0f9c-e26b-11ed-82db-abc3ceb7826d"),
				MasterCallID: uuid.FromStringOrNil("9052a47c-e26b-11ed-989e-4f387b975c9f"),
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9052a47c-e26b-11ed-989e-4f387b975c9f"),
				},
				Status: call.StatusHangup,
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.groupcall.MasterCallID).Return(tt.responseCall, nil)

			// hangup
			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.groupcall.ID, groupcall.StatusHangup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcall.ID).Return(tt.groupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.groupcall.CustomerID, groupcall.EventTypeGroupcallHangup, tt.groupcall)

			res, err := h.hangupRingMethodLinear(ctx, tt.groupcall)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.groupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.groupcall, res)
			}
		})
	}
}

func Test_hangupRingMethodLinear_groupcall_has_invalid_dialindex(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall
	}{
		{
			name: "dial index is equal len",

			groupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("fb09d8bc-e26b-11ed-9257-4f3c882e1db1"),
				Destinations: []commonaddress.Address{
					{
						Type: commonaddress.TypeTel,
					},
					{
						Type: commonaddress.TypeTel,
					},
				},
				DialIndex: 1,
			},
		},
		{
			name: "dial index is over len",

			groupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("fb09d8bc-e26b-11ed-9257-4f3c882e1db1"),
				Destinations: []commonaddress.Address{
					{
						Type: commonaddress.TypeTel,
					},
					{
						Type: commonaddress.TypeTel,
					},
				},
				DialIndex: 2,
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

			// hangup
			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.groupcall.ID, groupcall.StatusHangup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcall.ID).Return(tt.groupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.groupcall.CustomerID, groupcall.EventTypeGroupcallHangup, tt.groupcall)

			res, err := h.hangupRingMethodLinear(ctx, tt.groupcall)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.groupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.groupcall, res)
			}
		})
	}
}

func Test_HangupGroupcall(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("1e3801c6-e44a-11ed-bbee-230bc52a8c1b"),

			responseGroupcall: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("1e3801c6-e44a-11ed-bbee-230bc52a8c1b"),
				CallCount:  0,
				RingMethod: groupcall.RingMethodRingAll,
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallDecreaseGroupcallCount(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			// Hangup
			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.responseGroupcall.ID, groupcall.StatusHangup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallHangup, tt.responseGroupcall)

			res, err := h.HangupGroupcall(ctx, tt.id)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_HangupCall_Ringall(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("4b3ad8d2-f7b6-4d8f-868b-364c25c18f6b"),

			responseGroupcall: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("4b3ad8d2-f7b6-4d8f-868b-364c25c18f6b"),
				CallCount:  0,
				RingMethod: groupcall.RingMethodRingAll,
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallDecreaseCallCount(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			// Hangup
			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.responseGroupcall.ID, groupcall.StatusHangup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallHangup, tt.responseGroupcall)

			res, err := h.HangupCall(ctx, tt.id)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_callHangupLinear(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("4b3ad8d2-f7b6-4d8f-868b-364c25c18f6b"),

			responseGroupcall: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("4b3ad8d2-f7b6-4d8f-868b-364c25c18f6b"),
				CallCount:  0,
				RingMethod: groupcall.RingMethodRingAll,
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallDecreaseCallCount(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			// Hangup
			mockDB.EXPECT().GroupcallSetStatus(ctx, tt.responseGroupcall.ID, groupcall.StatusHangup).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallHangup, tt.responseGroupcall)

			res, err := h.HangupCall(ctx, tt.id)
			if err != nil {
				t.Errorf("wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseGroupcall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}
