package groupcallhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_AnswerCall(t *testing.T) {

	tests := []struct {
		name string

		groupcallID  uuid.UUID
		answerCallID uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			groupcallID:  uuid.FromStringOrNil("7669f00e-bb26-11ed-a4c3-bf62864985db"),
			answerCallID: uuid.FromStringOrNil("769f39e4-bb26-11ed-928d-1309c50d6617"),

			responseGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("7669f00e-bb26-11ed-a4c3-bf62864985db"),
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				AnswerCallID: uuid.FromStringOrNil("769f39e4-bb26-11ed-928d-1309c50d6617"),
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

			mockDB.EXPECT().GroupcallSetAnswerCallID(ctx, tt.groupcallID, tt.answerCallID).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallProgressing, tt.responseGroupcall)

			if errAnswer := h.AnswerCall(ctx, tt.groupcallID, tt.answerCallID); errAnswer != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errAnswer)
			}
		})
	}
}

func Test_AnswerGroupcall(t *testing.T) {

	tests := []struct {
		name string

		groupcallID       uuid.UUID
		answerGroupcallID uuid.UUID

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "have no master groupcall id",

			groupcallID:       uuid.FromStringOrNil("811d3277-a22c-4732-8b81-c8e5077876b5"),
			answerGroupcallID: uuid.FromStringOrNil("26e62358-9898-4920-a871-42cd14afcb8a"),

			responseGroupcall: &groupcall.Groupcall{
				ID:                uuid.FromStringOrNil("811d3277-a22c-4732-8b81-c8e5077876b5"),
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerGroupcallID: uuid.FromStringOrNil("26e62358-9898-4920-a871-42cd14afcb8a"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("5f935eb4-4958-47d4-aa47-a76e4c14bb4f"),
					uuid.FromStringOrNil("71692b53-f6d7-4da3-b3c8-4c099b7377e5"),
				},
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("1ff39315-5cb7-4f33-a1cf-f326b08fab22"),
					uuid.FromStringOrNil("cf4ebfff-bb33-4e97-9581-7aca3cfca352"),
				},
			},
		},
		{
			name: "have master groupcall id",

			groupcallID:       uuid.FromStringOrNil("83b6b795-aa1c-4ec5-a7ec-1af8ae8c14e3"),
			answerGroupcallID: uuid.FromStringOrNil("0e58d8fb-7dba-4d43-9226-76c4dd1c256e"),

			responseGroupcall: &groupcall.Groupcall{
				ID:                uuid.FromStringOrNil("83b6b795-aa1c-4ec5-a7ec-1af8ae8c14e3"),
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerGroupcallID: uuid.FromStringOrNil("0e58d8fb-7dba-4d43-9226-76c4dd1c256e"),
				MasterGroupcallID: uuid.FromStringOrNil("b78ed0c3-9266-4844-852b-12b645f4ebe1"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("3fd11c99-4555-4fa6-ab24-ada26dead9ff"),
					uuid.FromStringOrNil("95a5bdb9-9c41-4451-9199-74a8c3c2102a"),
				},
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("414c5882-a535-4204-93e3-977b76e4f24d"),
					uuid.FromStringOrNil("6a795ceb-7a83-4f39-88af-4210df125570"),
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

			mockDB.EXPECT().GroupcallSetAnswerGroupcallID(ctx, tt.groupcallID, tt.answerGroupcallID).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallProgressing, tt.responseGroupcall)

			if tt.responseGroupcall.MasterGroupcallID != uuid.Nil {
				mockReq.EXPECT().CallV1GroupcallUpdateAnswerGroupcallID(ctx, tt.responseGroupcall.MasterCallID, tt.responseGroupcall.ID).Return(&groupcall.Groupcall{}, nil)
			} else {
				// hangup others
				for _, groupcallID := range tt.responseGroupcall.GroupcallIDs {
					mockReq.EXPECT().CallV1GroupcallHangupOthers(ctx, groupcallID).Return(nil)
				}

				for _, callID := range tt.responseGroupcall.CallIDs {
					if callID == tt.responseGroupcall.AnswerCallID {
						continue
					}
					mockReq.EXPECT().CallV1CallHangup(ctx, callID).Return(&call.Call{}, nil)
				}
			}

			res, err := h.AnswerGroupcall(ctx, tt.groupcallID, tt.answerGroupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if res != tt.responseGroupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_answerCommon(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall
	}{
		{
			name: "have no master groupcall id",

			groupcall: &groupcall.Groupcall{
				ID:                uuid.FromStringOrNil("16dd70b2-02b6-4807-aed1-b0b16b21f02d"),
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerGroupcallID: uuid.FromStringOrNil("7d5d6cec-bd33-41e9-b23d-7909b6bdfd5d"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("445294da-6442-4140-9b6c-e37593b9035c"),
					uuid.FromStringOrNil("109d5237-718c-4716-af98-da5c08c3884e"),
				},
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("33bf2f3a-3a90-4dee-b3a5-eda64a1fef2b"),
					uuid.FromStringOrNil("51671580-ee27-4de7-852d-99109d27179b"),
				},
			},
		},
		{
			name: "have master groupcall id",

			groupcall: &groupcall.Groupcall{
				ID:                uuid.FromStringOrNil("9c89b1c5-fc4e-41c5-85f4-314c15c3bfea"),
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerGroupcallID: uuid.FromStringOrNil("f1d1a43e-833c-447d-933b-004239b97360"),
				MasterGroupcallID: uuid.FromStringOrNil("49bee46a-8741-4e4a-8287-dffa85375398"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("5cabddd0-21ad-41b1-a37b-ccfb7f825063"),
					uuid.FromStringOrNil("98462bdd-0f33-4aaf-8ebb-eb7123639fcd"),
				},
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("cc790772-5fb7-43dd-b8f9-5e4861136cac"),
					uuid.FromStringOrNil("95d03663-9235-4ef4-a667-e528f8989018"),
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

			if tt.groupcall.MasterGroupcallID != uuid.Nil {
				mockReq.EXPECT().CallV1GroupcallUpdateAnswerGroupcallID(ctx, tt.groupcall.MasterCallID, tt.groupcall.ID).Return(&groupcall.Groupcall{}, nil)
			} else {
				// hangup others
				for _, groupcallID := range tt.groupcall.GroupcallIDs {
					mockReq.EXPECT().CallV1GroupcallHangupOthers(ctx, groupcallID).Return(nil)
				}

				for _, callID := range tt.groupcall.CallIDs {
					if callID == tt.groupcall.AnswerCallID {
						continue
					}
					mockReq.EXPECT().CallV1CallHangup(ctx, callID).Return(&call.Call{}, nil)
				}
			}

			res, err := h.answerCommon(ctx, tt.groupcall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if res != tt.groupcall {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.groupcall, res)
			}
		})
	}
}
