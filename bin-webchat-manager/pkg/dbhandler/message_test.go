package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/pkg/cachehandler"
)

func Test_MessageCreate(t *testing.T) {

	tests := []struct {
		name string

		message *message.Message

		responseCurTime *time.Time
		expectRes       *message.Message
	}{
		{
			name: "normal",
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("8baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				SessionID: uuid.FromStringOrNil("8c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Direction: message.DirectionInbound,
				Status:    message.StatusSent,
				Text:      "hello there",
			},

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("8baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				SessionID: uuid.FromStringOrNil("8c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Direction: message.DirectionInbound,
				Status:    message.StatusSent,
				Text:      "hello there",
				TMCreate:  timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
				TMDelete:  nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageList(t *testing.T) {
	type test struct {
		name string
		data []*message.Message

		size    uint64
		token   string
		filters map[message.Field]any

		responseCurtime *time.Time
		expectRes       []*message.Message
	}

	sessionID := uuid.FromStringOrNil("9c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1")
	customerID := uuid.FromStringOrNil("9caabbb2-6b7c-11f0-9f8f-7307c1d1f7ea")

	tests := []test{
		{
			name: "normal",
			data: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9d8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
						CustomerID: customerID,
					},
					SessionID: sessionID,
					Direction: message.DirectionInbound,
					Status:    message.StatusSent,
					Text:      "list test message",
				},
			},

			size:    10,
			token:   "",
			filters: map[message.Field]any{message.FieldSessionID: sessionID, message.FieldDeleted: false},

			responseCurtime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9d8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
						CustomerID: customerID,
					},
					SessionID: sessionID,
					Direction: message.DirectionInbound,
					Status:    message.StatusSent,
					Text:      "list test message",
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

			for _, m := range tt.data {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurtime)
				mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				if err := h.MessageCreate(ctx, m); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.NewUtilHandler().TimeGetCurTime()).AnyTimes()
			res, err := h.MessageList(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != len(tt.expectRes) {
				t.Errorf("Wrong match. expect len: %d, got len: %d, res: %v", len(tt.expectRes), len(res), res)
			}
		})
	}
}

func Test_MessageDelete(t *testing.T) {

	tests := []struct {
		name string

		message *message.Message

		id uuid.UUID

		responseCurTime *time.Time
	}{
		{
			name: "normal",
			message: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ab8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("abaabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				SessionID: uuid.FromStringOrNil("ac8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Direction: message.DirectionOutbound,
				Status:    message.StatusSent,
				Text:      "delete me",
			},

			id: uuid.FromStringOrNil("ab8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.MessageDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.MessageGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == nil {
				t.Errorf("Wrong match. expect: non-nil TMDelete, got: nil")
			}
		})
	}
}
