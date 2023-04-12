package groupcallhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_HangingupOthers(t *testing.T) {

	tests := []struct {
		name string

		groupcall *groupcall.Groupcall

		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",

			groupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("da99d4e8-d905-11ed-8a4c-a72c1eb8b80f"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("db0412c2-d905-11ed-b350-272f54423bec"),
					uuid.FromStringOrNil("db2f3c86-d905-11ed-aa9e-d7752d9d4d3f"),
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

			for _, callID := range tt.groupcall.CallIDs {
				if callID == tt.groupcall.AnswerCallID {
					continue
				}

				mockReq.EXPECT().CallV1CallHangup(ctx, callID).Return(&call.Call{}, nil)
			}

			if errHangup := h.HangingupOthers(ctx, tt.groupcall); errHangup != nil {
				t.Errorf("wrong match.\nexpect: nil\ngot: %v", errHangup)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
