package groupcallhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_AnswerCall_cov2_errors(t *testing.T) {
	id := uuid.FromStringOrNil("7669f00e-bb26-11ed-a4c3-bf62864985db")
	callID := uuid.FromStringOrNil("769f39e4-bb26-11ed-928d-1309c50d6617")
	masterCallID := uuid.FromStringOrNil("b78ed0c3-9266-4844-852b-12b645f4ebe1")

	tests := []struct {
		name string

		setErr error
		gc     *groupcall.Groupcall
		reqErr error

		expectErr string
	}{
		{name: "set answer call id error", setErr: errors.New("set"), expectErr: "Could not update the answer call id"},
		{
			name:      "unsupported answer method",
			gc:        &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, AnswerMethod: groupcall.AnswerMethod("none")},
			expectErr: "could not handle the answerCommon process correctly: unsupported answer method",
		},
		{
			name:      "master update error",
			gc:        &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, MasterGroupcallID: masterCallID, MasterCallID: masterCallID},
			reqErr:    errors.New("req"),
			expectErr: "could not handle the answerCommon process correctly: could not update the answer groupcall id: req",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &groupcallHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetAnswerCallID(ctx, id, callID).Return(tt.setErr)
			if tt.setErr == nil {
				mockDB.EXPECT().GroupcallGet(ctx, id).Return(tt.gc, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.gc.CustomerID, groupcall.EventTypeGroupcallProgressing, tt.gc)
			}
			if tt.reqErr != nil {
				mockReq.EXPECT().CallV1GroupcallUpdateAnswerGroupcallID(ctx, gomock.Any(), id).Return(nil, tt.reqErr)
			}

			err := h.AnswerCall(ctx, id, callID)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_AnswerGroupcall_cov2_errors(t *testing.T) {
	id := uuid.FromStringOrNil("811d3277-a22c-4732-8b81-c8e5077876b5")
	answerID := uuid.FromStringOrNil("26e62358-9898-4920-a871-42cd14afcb8a")

	tests := []struct {
		name string

		setErr error
		gc     *groupcall.Groupcall

		expectErr string
	}{
		{name: "set answer groupcall id error", setErr: errors.New("set"), expectErr: "Could not update the answer call id"},
		{
			name:      "unsupported answer method",
			gc:        &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, AnswerMethod: groupcall.AnswerMethod("none")},
			expectErr: "could not handle the common answer: unsupported answer method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &groupcallHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallSetAnswerGroupcallID(ctx, id, answerID).Return(tt.setErr)
			if tt.setErr == nil {
				mockDB.EXPECT().GroupcallGet(ctx, id).Return(tt.gc, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.gc.CustomerID, groupcall.EventTypeGroupcallProgressing, tt.gc)
			}

			res, err := h.AnswerGroupcall(ctx, id, answerID)
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v / %v", tt.expectErr, res, err)
			}
		})
	}
}
