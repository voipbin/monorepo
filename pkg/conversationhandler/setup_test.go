package conversationhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
)

func Test_Setup(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType conversation.ReferenceType
	}{
		{
			"line messages",

			uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
			conversation.ReferenceTypeLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}

			ctx := context.Background()

			switch tt.referenceType {
			case conversation.ReferenceTypeLine:
				mockLine.EXPECT().Setup(ctx, tt.customerID).Return(nil)
			}

			err := h.Setup(ctx, tt.customerID, tt.referenceType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Setup_error(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType conversation.ReferenceType
	}{
		{
			"reference type none",

			uuid.FromStringOrNil("7b1034a8-e6ef-11ec-9e9d-c3f3e36741ac"),
			conversation.ReferenceTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}

			ctx := context.Background()

			err := h.Setup(ctx, tt.customerID, tt.referenceType)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

		})
	}
}
