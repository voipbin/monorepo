package chatbotcallhandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
)

func Test_ProcessStart(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall

		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			"normal",

			&chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("6ed69462-a705-11ed-a47b-cfb979f9f07d"),
					CustomerID: uuid.FromStringOrNil("6f12ea52-a705-11ed-86d3-8b796a5da603"),
				},
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("6f69db50-a705-11ed-bc35-177b3c1673d4"),
				Language:      "en-US",
			},

			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("6f40a2c6-a705-11ed-8981-d78afab8acba"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
			}

			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.chatbotcall.CustomerID, tmtranscribe.ReferenceTypeCall, tt.chatbotcall.ReferenceID, tt.chatbotcall.Language, tmtranscribe.DirectionIn).Return(tt.responseTranscribe, nil)
			mockDB.EXPECT().ChatbotcallUpdateStatusProgressing(ctx, tt.chatbotcall.ID, tt.responseTranscribe.ID).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.chatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.chatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallProgressing, tt.chatbotcall)

			res, err := h.ProcessStart(ctx, tt.chatbotcall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.chatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.chatbotcall, res)
			}
		})
	}
}

func Test_ProcessEnd(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall

		responseTranscribe *tmtranscribe.Transcribe
	}{
		{
			"normal",

			&chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a7c462f8-a706-11ed-9461-cbd173399722"),
				},
				ConfbridgeID: uuid.FromStringOrNil("fe18ea48-e12d-43cb-8b40-48caeed6d67b"),
				TranscribeID: uuid.FromStringOrNil("a7f1d814-a706-11ed-9af7-3f37982d3546"),
			},

			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("a7f1d814-a706-11ed-9af7-3f37982d3546"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
			}

			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscribeStop(ctx, tt.chatbotcall.TranscribeID).Return(&tmtranscribe.Transcribe{}, nil)
			mockDB.EXPECT().ChatbotcallUpdateStatusEnd(ctx, tt.chatbotcall.ID).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.chatbotcall, nil)
			mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.chatbotcall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.chatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallEnd, tt.chatbotcall)

			res, err := h.ProcessEnd(ctx, tt.chatbotcall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.chatbotcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.chatbotcall, res)
			}
		})
	}
}
