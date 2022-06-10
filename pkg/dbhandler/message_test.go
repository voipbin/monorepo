package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/cachehandler"
)

func Test_MessageCreate(t *testing.T) {

	tests := []struct {
		name string

		message *message.Message
	}{
		{
			"normal",

			&message.Message{
				ID:             uuid.FromStringOrNil("19c162d4-e4a2-11ec-a3ce-ef751a8980e7"),
				CustomerID:     uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Status:         message.StatusReceived,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				SourceID:       "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Data:           []byte(`{"type":"message","message":{"type":"text","id":"16179377757574","text":"Hi"},"webhookEventId":"01G4C3RCEBKMDJWAQ5B6Z627V3","deliveryContext":{"isRedelivery":false},"timestamp":1653969007017,"source":{"type":"user","userId":"Ud871bcaf7c3ad13d2a0b0d78a42a287f"},"replyToken":"365b72b47e374bd489f9a9207115dd34","mode":"active"}`),
				TMCreate:       "2022-04-18 03:22:17.995000",
				TMUpdate:       "2022-04-18 03:22:17.995000",
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessageGet(gomock.Any(), tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.message) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.message, res)
			}
		})
	}
}

func Test_MessageGetsByConversationID(t *testing.T) {

	tests := []struct {
		name string

		messages []*message.Message

		conversationID uuid.UUID
		token          string
		limit          uint64
	}{
		{
			"normal",

			[]*message.Message{
				{
					ID:             uuid.FromStringOrNil("b0f40ae2-e4a4-11ec-b2f2-9f29af0582dc"),
					CustomerID:     uuid.FromStringOrNil("b11b373e-e4a4-11ec-b28e-0f4453fab505"),
					ConversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					SourceID:       "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Data:           []byte(`{"type":"message","message":{"type":"text","id":"16179377757574","text":"Hi"},"webhookEventId":"01G4C3RCEBKMDJWAQ5B6Z627V3","deliveryContext":{"isRedelivery":false},"timestamp":1653969007017,"source":{"type":"user","userId":"Ud871bcaf7c3ad13d2a0b0d78a42a287f"},"replyToken":"365b72b47e374bd489f9a9207115dd34","mode":"active"}`),
					TMCreate:       "2022-04-18 03:22:17.995000",
					TMUpdate:       "2022-04-18 03:22:17.995000",
					TMDelete:       DefaultTimeStamp,
				},
				{
					ID:             uuid.FromStringOrNil("1f4dc5fa-e4a5-11ec-9ee3-1f32b34259d3"),
					CustomerID:     uuid.FromStringOrNil("b11b373e-e4a4-11ec-b28e-0f4453fab505"),
					ConversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					SourceID:       "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Data:           []byte(`{"type":"message","message":{"type":"text","id":"16179377757574","text":"Hi"},"webhookEventId":"01G4C3RCEBKMDJWAQ5B6Z627V3","deliveryContext":{"isRedelivery":false},"timestamp":1653969007017,"source":{"type":"user","userId":"Ud871bcaf7c3ad13d2a0b0d78a42a287f"},"replyToken":"365b72b47e374bd489f9a9207115dd34","mode":"active"}`),
					TMCreate:       "2022-04-18 02:22:17.995000",
					TMUpdate:       "2022-04-18 02:22:17.995000",
					TMDelete:       DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
			"2022-05-18 04:22:17.995000",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, m := range tt.messages {
				mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any())
				if err := h.MessageCreate(ctx, m); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.MessageGetsByConversationID(ctx, tt.conversationID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.messages) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.messages, res)
			}
		})
	}
}
