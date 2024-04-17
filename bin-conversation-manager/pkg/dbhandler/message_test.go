package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/cachehandler"
)

func Test_MessageCreate(t *testing.T) {

	tests := []struct {
		name string

		message *message.Message

		responseCurTime string
		expectRes       *message.Message
	}{
		{
			name: "normal",

			message: &message.Message{
				ID:             uuid.FromStringOrNil("19c162d4-e4a2-11ec-a3ce-ef751a8980e7"),
				CustomerID:     uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Direction:      message.DirectionIncoming,
				Status:         message.StatusReceived,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TransactionID:  "207b7274-f175-11ec-acf9-73a933332479",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				},
				Text:   "Hello world",
				Medias: []media.Media{},
			},

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: &message.Message{
				ID:             uuid.FromStringOrNil("19c162d4-e4a2-11ec-a3ce-ef751a8980e7"),
				CustomerID:     uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Direction:      message.DirectionIncoming,
				Status:         message.StatusReceived,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TransactionID:  "207b7274-f175-11ec-acf9-73a933332479",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				},
				Text:     "Hello world",
				Medias:   []media.Media{},
				TMCreate: "2022-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// message get
			mockCache.EXPECT().MessageGet(gomock.Any(), tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil)
			resGet, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(resGet, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.message, resGet)
			}

			// message get by transaction_id
			resTransaction, err := h.MessageGetsByTransactionID(ctx, tt.message.TransactionID, utilhandler.TimeGetCurTime(), 10)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(resTransaction, []*message.Message{tt.expectRes}) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, resTransaction)
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

		responseCurTime string
		expectRes       []*message.Message
	}{
		{
			name: "normal",

			messages: []*message.Message{
				{
					ID:             uuid.FromStringOrNil("b0f40ae2-e4a4-11ec-b2f2-9f29af0582dc"),
					CustomerID:     uuid.FromStringOrNil("b11b373e-e4a4-11ec-b28e-0f4453fab505"),
					ConversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeLine,
						Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					},
					Text:   "hello world",
					Medias: []media.Media{},
				},
				{
					ID:             uuid.FromStringOrNil("1f4dc5fa-e4a5-11ec-9ee3-1f32b34259d3"),
					CustomerID:     uuid.FromStringOrNil("b11b373e-e4a4-11ec-b28e-0f4453fab505"),
					ConversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeLine,
						Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					},
					Text:   "This is test",
					Medias: []media.Media{},
				},
			},

			conversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
			token:          "2022-05-18 04:22:17.995000",
			limit:          100,

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: []*message.Message{
				{
					ID:             uuid.FromStringOrNil("b0f40ae2-e4a4-11ec-b2f2-9f29af0582dc"),
					CustomerID:     uuid.FromStringOrNil("b11b373e-e4a4-11ec-b28e-0f4453fab505"),
					ConversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeLine,
						Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					},
					Text:     "hello world",
					Medias:   []media.Media{},
					TMCreate: "2022-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					ID:             uuid.FromStringOrNil("1f4dc5fa-e4a5-11ec-9ee3-1f32b34259d3"),
					CustomerID:     uuid.FromStringOrNil("b11b373e-e4a4-11ec-b28e-0f4453fab505"),
					ConversationID: uuid.FromStringOrNil("b29dd422-e4a4-11ec-a381-37d969f9b237"),
					Status:         message.StatusReceived,
					ReferenceType:  conversation.ReferenceTypeLine,
					ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					Source: &commonaddress.Address{
						Type:   commonaddress.TypeLine,
						Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
					},
					Text:     "This is test",
					Medias:   []media.Media{},
					TMCreate: "2022-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, m := range tt.messages {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any())
				if err := h.MessageCreate(ctx, m); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.MessageGetsByConversationID(ctx, tt.conversationID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_MessageUpdateStatus(t *testing.T) {

	tests := []struct {
		name    string
		message *message.Message

		status message.Status

		responseCurTime string
		expectRes       *message.Message
	}{
		{
			name: "test normal",
			message: &message.Message{
				ID:         uuid.FromStringOrNil("fc67b82c-a2a3-11ec-970f-1f9f06c64b70"),
				CustomerID: uuid.FromStringOrNil("3f7a4c24-a2a4-11ec-b26e-3f8d47c2b450"),
				Status:     message.StatusSending,
			},

			status: message.StatusSent,

			responseCurTime: "2021-02-26 18:26:49.000",
			expectRes: &message.Message{
				ID:         uuid.FromStringOrNil("fc67b82c-a2a3-11ec-970f-1f9f06c64b70"),
				CustomerID: uuid.FromStringOrNil("3f7a4c24-a2a4-11ec-b26e-3f8d47c2b450"),
				Status:     message.StatusSent,

				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: "2021-02-26 18:26:49.000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageUpdateStatus(ctx, tt.message.ID, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_MessageDelete(t *testing.T) {

	tests := []struct {
		name    string
		message *message.Message

		responseCurTime string
		expectRes       *message.Message
	}{
		{
			name: "normal",
			message: &message.Message{
				ID:             uuid.FromStringOrNil("4292410e-f1d8-11ec-b03e-639c5da6a05a"),
				CustomerID:     uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Status:         message.StatusReceived,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TransactionID:  "207b7274-f175-11ec-acf9-73a933332479",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				},
				Text:   "Hello world",
				Medias: []media.Media{},
			},

			responseCurTime: "2021-02-26 18:26:49.000",
			expectRes: &message.Message{
				ID:             uuid.FromStringOrNil("4292410e-f1d8-11ec-b03e-639c5da6a05a"),
				CustomerID:     uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Status:         message.StatusReceived,
				ReferenceType:  conversation.ReferenceTypeLine,
				ReferenceID:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				TransactionID:  "207b7274-f175-11ec-acf9-73a933332479",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				},
				Text:     "Hello world",
				Medias:   []media.Media{},
				TMCreate: "2021-02-26 18:26:49.000",
				TMUpdate: "2021-02-26 18:26:49.000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageDelete(ctx, tt.message.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
