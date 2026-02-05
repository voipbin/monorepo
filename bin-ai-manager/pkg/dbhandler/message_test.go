package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_MessageCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name string

		message *message.Message

		responseCurTime *time.Time
		expectRes       *message.Message
	}{
		{
			name: "valid message",

			message: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d5df8eac-f22b-11ef-b88e-7f62eefdf1ca"),
					CustomerID: uuid.FromStringOrNil("2093691e-f22c-11ef-bf60-a717f01b92a4"),
				},
				AIcallID: uuid.FromStringOrNil("d6555614-f22b-11ef-96c2-e7d5f61b54dd"),

				Role:    message.RoleUser,
				Content: "Hello",

				ToolCalls: []message.ToolCall{
					{
						ID:   "44e89598-9324-11f0-aa28-1f3b222aa599",
						Type: message.ToolTypeFunction,
						Function: message.FunctionCall{
							Name:      "get_current_weather",
							Arguments: `{"location": "Boston, MA", "unit": "celsius"}`,
						},
					},
				},
				ToolCallID: "6798165e-9324-11f0-91a4-c7ebb2a64dfd",
			},

			responseCurTime: curTime,
			expectRes: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d5df8eac-f22b-11ef-b88e-7f62eefdf1ca"),
					CustomerID: uuid.FromStringOrNil("2093691e-f22c-11ef-bf60-a717f01b92a4"),
				},
				AIcallID: uuid.FromStringOrNil("d6555614-f22b-11ef-96c2-e7d5f61b54dd"),

				Role:    message.RoleUser,
				Content: "Hello",

				ToolCalls: []message.ToolCall{
					{
						ID:   "44e89598-9324-11f0-aa28-1f3b222aa599",
						Type: message.ToolTypeFunction,
						Function: message.FunctionCall{
							Name:      "get_current_weather",
							Arguments: `{"location": "Boston, MA", "unit": "celsius"}`,
						},
					},
				},
				ToolCallID: "6798165e-9324-11f0-91a4-c7ebb2a64dfd",

				TMCreate: curTime,
				TMDelete: nil,
			},
		},
		{
			name: "empty content",

			message: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d62e7a58-f22b-11ef-8edc-9b57d94ff8fc"),
				},
				AIcallID: uuid.FromStringOrNil("20b4c03c-f22c-11ef-abe7-3b10f3525941"),
			},

			responseCurTime: curTime,
			expectRes: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d62e7a58-f22b-11ef-8edc-9b57d94ff8fc"),
				},
				AIcallID:  uuid.FromStringOrNil("20b4c03c-f22c-11ef-abe7-3b10f3525941"),
				ToolCalls: nil,
				TMCreate:  curTime,
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
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())

			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			expectRes := []*message.Message{tt.expectRes}
			resGets, err := h.MessageList(ctx, 100, DefaultTimeStamp, map[message.Field]any{
				message.FieldAIcallID: tt.message.AIcallID,
			})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(expectRes, resGets) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, resGets)
			}
		})
	}
}
