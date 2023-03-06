package groupcallhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Answer(t *testing.T) {

	tests := []struct {
		name string

		groupcallID  uuid.UUID
		answerCallID uuid.UUID

		responseGroupcall *groupcall.Groupcall
		responseGets      []*call.Call

		expectGroupcall *groupcall.Groupcall

		expectRes []*call.Call
	}{
		{
			name: "normal",

			groupcallID:  uuid.FromStringOrNil("7669f00e-bb26-11ed-a4c3-bf62864985db"),
			answerCallID: uuid.FromStringOrNil("769f39e4-bb26-11ed-928d-1309c50d6617"),

			responseGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("7669f00e-bb26-11ed-a4c3-bf62864985db"),
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
			},

			expectGroupcall: &groupcall.Groupcall{
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

			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockDB.EXPECT().GroupcallUpdate(ctx, tt.expectGroupcall).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.groupcallID).Return(tt.expectGroupcall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, groupcall.EventTypeGroupcallAnswered, tt.expectGroupcall)

			if errAnswer := h.Answer(ctx, tt.groupcallID, tt.answerCallID); errAnswer != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errAnswer)
			}
		})
	}
}
