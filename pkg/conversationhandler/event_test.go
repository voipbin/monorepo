package conversationhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/messagehandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/smshandler"
)

func Test_Event(t *testing.T) {

	tests := []struct {
		name string

		referenceType conversation.ReferenceType
		data          []byte

		responseEvent        []*message.Message
		responseConversation *conversation.Conversation
	}{
		{
			"normal",

			conversation.ReferenceTypeMessage,
			[]byte(`{"id":"5e65e04e-f12d-11ec-b951-53cf815f86a4"}`),

			[]*message.Message{
				{
					ID:     uuid.FromStringOrNil("c4fd2a56-f12d-11ec-b443-1f1133008bfc"),
					Source: &commonaddress.Address{},
				},
			},
			&conversation.Conversation{
				ID: uuid.FromStringOrNil("f45df2d0-f12d-11ec-bd7f-2f3e6d9a6218"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSMS := smshandler.NewMockSMSHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,

				messageHandler: mockMessage,
				smsHandler:     mockSMS,
			}

			ctx := context.Background()

			mockSMS.EXPECT().Event(ctx, tt.data).Return(tt.responseEvent, &commonaddress.Address{}, nil)
			mockMessage.EXPECT().GetsByTransactionID(ctx, tt.responseEvent[0].TransactionID, gomock.Any(), uint64(10)).Return([]*message.Message{}, nil)
			for _, tmp := range tt.responseEvent {

				mockDB.EXPECT().ConversationGetByReferenceInfo(ctx, tmp.ReferenceType, gomock.Any()).Return(tt.responseConversation, nil)
				mockMessage.EXPECT().Create(
					ctx,
					tt.responseConversation.CustomerID,
					tt.responseConversation.ID,
					message.StatusReceived,
					conversation.ReferenceTypeMessage,
					tmp.ReferenceID,
					tmp.ID.String(),
					tmp.Source,
					tmp.Text,
					tmp.Medias,
				).Return(tmp, nil)
			}

			if err := h.Event(ctx, tt.referenceType, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_eventSMS(t *testing.T) {

	tests := []struct {
		name string

		data []byte

		responseEvent        []*message.Message
		responseConversation *conversation.Conversation
	}{
		{
			"normal",

			[]byte(`{"id":"1fe68e20-f12f-11ec-84fe-03665484eeb6"}`),

			[]*message.Message{
				{
					ID:     uuid.FromStringOrNil("20ab5674-f12f-11ec-9809-9ba9b288fb2c"),
					Source: &commonaddress.Address{},
				},
			},
			&conversation.Conversation{
				ID: uuid.FromStringOrNil("20d80048-f12f-11ec-8f8d-affa9735f9de"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSMS := smshandler.NewMockSMSHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,

				messageHandler: mockMessage,
				smsHandler:     mockSMS,
			}

			ctx := context.Background()

			mockSMS.EXPECT().Event(ctx, tt.data).Return(tt.responseEvent, &commonaddress.Address{}, nil)
			mockMessage.EXPECT().GetsByTransactionID(ctx, tt.responseEvent[0].TransactionID, gomock.Any(), uint64(10)).Return([]*message.Message{}, nil)

			for _, tmp := range tt.responseEvent {

				mockDB.EXPECT().ConversationGetByReferenceInfo(ctx, tmp.ReferenceType, gomock.Any()).Return(tt.responseConversation, nil)
				mockMessage.EXPECT().Create(
					ctx,
					tt.responseConversation.CustomerID,
					tt.responseConversation.ID,
					message.StatusReceived,
					conversation.ReferenceTypeMessage,
					tmp.ReferenceID,
					tmp.ID.String(),
					tmp.Source,
					tmp.Text,
					tmp.Medias,
				).Return(tmp, nil)
			}

			if err := h.eventSMS(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
