package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("19c162d4-e4a2-11ec-a3ce-ef751a8980e7"),
					CustomerID: uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				},
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Direction:      message.DirectionIncoming,
				Status:         message.StatusDone,
				ReferenceType:  message.ReferenceTypeLine,
				ReferenceID:    uuid.FromStringOrNil("207b7274-f175-11ec-acf9-73a933332479"),
				TransactionID:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Text:           "Hello world",
				Medias:         []media.Media{},
			},

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("19c162d4-e4a2-11ec-a3ce-ef751a8980e7"),
					CustomerID: uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				},
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Direction:      message.DirectionIncoming,
				Status:         message.StatusDone,
				ReferenceType:  message.ReferenceTypeLine,
				ReferenceID:    uuid.FromStringOrNil("207b7274-f175-11ec-acf9-73a933332479"),
				TransactionID:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Text:           "Hello world",
				Medias:         []media.Media{},
				TMCreate:       "2022-04-18 03:22:17.995000",
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
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

func Test_MessageGets(t *testing.T) {

	tests := []struct {
		name     string
		messages []*message.Message

		token   string
		limit   uint64
		filters map[message.Field]any

		responseCurTime string
		expectRes       []*message.Message
	}{
		{
			name: "normal",
			messages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c4b65d46-1bce-11f0-aecf-6f12ebb97849"),
						CustomerID: uuid.FromStringOrNil("c4f26c14-1bce-11f0-8e0c-13f9ca3df39e"),
					},
					ConversationID: uuid.FromStringOrNil("c51b50ca-1bce-11f0-8c4c-db38779c786d"),
					Status:         message.StatusDone,
					ReferenceType:  message.ReferenceTypeLine,
					ReferenceID:    uuid.FromStringOrNil("c548ef9e-1bce-11f0-801c-1f5a90192b72"),
					Text:           "hello world",
					Medias:         []media.Media{},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1f4dc5fa-e4a5-11ec-9ee3-1f32b34259d3"),
						CustomerID: uuid.FromStringOrNil("c4f26c14-1bce-11f0-8e0c-13f9ca3df39e"),
					},
					ConversationID: uuid.FromStringOrNil("c51b50ca-1bce-11f0-8c4c-db38779c786d"),
					Status:         message.StatusDone,
					ReferenceType:  message.ReferenceTypeLine,
					ReferenceID:    uuid.FromStringOrNil("c57b83be-1bce-11f0-904e-171ea4fa9d1b"),
					Text:           "This is test",
					Medias:         []media.Media{},
				},
			},

			token: "2022-05-18 04:22:17.995000",
			limit: 100,
			filters: map[message.Field]any{
				message.FieldConversationID: uuid.FromStringOrNil("c51b50ca-1bce-11f0-8c4c-db38779c786d"),
			},

			responseCurTime: "2022-04-18 03:22:17.995000",
			expectRes: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c4b65d46-1bce-11f0-aecf-6f12ebb97849"),
						CustomerID: uuid.FromStringOrNil("c4f26c14-1bce-11f0-8e0c-13f9ca3df39e"),
					},
					ConversationID: uuid.FromStringOrNil("c51b50ca-1bce-11f0-8c4c-db38779c786d"),
					Status:         message.StatusDone,
					ReferenceType:  message.ReferenceTypeLine,
					ReferenceID:    uuid.FromStringOrNil("c548ef9e-1bce-11f0-801c-1f5a90192b72"),
					Text:           "hello world",
					Medias:         []media.Media{},
					TMCreate:       "2022-04-18 03:22:17.995000",
					TMUpdate:       DefaultTimeStamp,
					TMDelete:       DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("1f4dc5fa-e4a5-11ec-9ee3-1f32b34259d3"),
						CustomerID: uuid.FromStringOrNil("c4f26c14-1bce-11f0-8e0c-13f9ca3df39e"),
					},
					ConversationID: uuid.FromStringOrNil("c51b50ca-1bce-11f0-8c4c-db38779c786d"),
					Status:         message.StatusDone,
					ReferenceType:  message.ReferenceTypeLine,
					ReferenceID:    uuid.FromStringOrNil("c57b83be-1bce-11f0-904e-171ea4fa9d1b"),
					Text:           "This is test",
					Medias:         []media.Media{},
					TMCreate:       "2022-04-18 03:22:17.995000",
					TMUpdate:       DefaultTimeStamp,
					TMDelete:       DefaultTimeStamp,
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

			res, err := h.MessageGets(ctx, tt.token, tt.limit, tt.filters)
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fc67b82c-a2a3-11ec-970f-1f9f06c64b70"),
					CustomerID: uuid.FromStringOrNil("3f7a4c24-a2a4-11ec-b26e-3f8d47c2b450"),
				},
				Status: message.StatusProgressing,
			},

			status: message.StatusDone,

			responseCurTime: "2021-02-26 18:26:49.000",
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fc67b82c-a2a3-11ec-970f-1f9f06c64b70"),
					CustomerID: uuid.FromStringOrNil("3f7a4c24-a2a4-11ec-b26e-3f8d47c2b450"),
				},
				Status: message.StatusDone,

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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4292410e-f1d8-11ec-b03e-639c5da6a05a"),
					CustomerID: uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				},
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Status:         message.StatusDone,
				ReferenceType:  message.ReferenceTypeLine,
				ReferenceID:    uuid.FromStringOrNil("3ce6f9e2-1bc5-11f0-8436-4b1f0f60ddf5"),
				TransactionID:  "207b7274-f175-11ec-acf9-73a933332479",
				Text:           "Hello world",
				Medias:         []media.Media{},
			},

			responseCurTime: "2021-02-26 18:26:49.000",
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4292410e-f1d8-11ec-b03e-639c5da6a05a"),
					CustomerID: uuid.FromStringOrNil("1a3cf002-e4a2-11ec-855c-9fdc2a6e37d3"),
				},
				ConversationID: uuid.FromStringOrNil("1a795984-e4a2-11ec-a8b0-37faa9ea3db2"),
				Status:         message.StatusDone,
				ReferenceType:  message.ReferenceTypeLine,
				ReferenceID:    uuid.FromStringOrNil("3ce6f9e2-1bc5-11f0-8436-4b1f0f60ddf5"),
				TransactionID:  "207b7274-f175-11ec-acf9-73a933332479",
				Text:           "Hello world",
				Medias:         []media.Media{},
				TMCreate:       "2021-02-26 18:26:49.000",
				TMUpdate:       "2021-02-26 18:26:49.000",
				TMDelete:       DefaultTimeStamp,
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

func Test_MessageUpdate(t *testing.T) {
	tests := []struct {
		name    string
		message *message.Message

		id    uuid.UUID
		field map[message.Field]any

		responseCurTime string
		expectRes       *message.Message
	}{
		{
			name: "normal",
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("64b74416-3413-11f0-809c-c30541817b53"),
					CustomerID: uuid.FromStringOrNil("010eac88-2199-11f0-ad61-671cf62bcc31"),
				},
			},

			id: uuid.FromStringOrNil("64b74416-3413-11f0-809c-c30541817b53"),
			field: map[message.Field]any{
				message.FieldConversationID: uuid.FromStringOrNil("65209fd8-3413-11f0-8792-9be4d9395c77"),
				message.FieldStatus:         message.StatusDone,
			},

			responseCurTime: "2020-04-18T03:22:17.995000",
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("64b74416-3413-11f0-809c-c30541817b53"),
					CustomerID: uuid.FromStringOrNil("010eac88-2199-11f0-ad61-671cf62bcc31"),
				},
				ConversationID: uuid.FromStringOrNil("65209fd8-3413-11f0-8792-9be4d9395c77"),
				Status:         message.StatusDone,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
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
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.MessageUpdate(ctx, tt.id, tt.field); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			res, err := h.MessageGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
