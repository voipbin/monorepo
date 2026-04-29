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
		{
			name: "with pipecatcall id and delivery status pending",

			message: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("4810ed7c-249a-11f1-9c4d-1be83fbb24d9"),
					CustomerID: uuid.FromStringOrNil("48541f10-249a-11f1-8c25-7763b87b41e2"),
				},
				AIcallID:       uuid.FromStringOrNil("48a04e08-249a-11f1-8e9f-1f53f0e2c9bf"),
				PipecatcallID:  uuid.FromStringOrNil("48ec3a36-249a-11f1-bc6c-2bd6e2e0fa7b"),
				DeliveryStatus: message.DeliveryStatusPending,

				Role:    message.RoleAssistant,
				Content: "queued reply",
			},

			responseCurTime: curTime,
			expectRes: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("4810ed7c-249a-11f1-9c4d-1be83fbb24d9"),
					CustomerID: uuid.FromStringOrNil("48541f10-249a-11f1-8c25-7763b87b41e2"),
				},
				AIcallID:       uuid.FromStringOrNil("48a04e08-249a-11f1-8e9f-1f53f0e2c9bf"),
				PipecatcallID:  uuid.FromStringOrNil("48ec3a36-249a-11f1-bc6c-2bd6e2e0fa7b"),
				DeliveryStatus: message.DeliveryStatusPending,

				Role:    message.RoleAssistant,
				Content: "queued reply",

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
			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime.Format(utilhandler.ISO8601Layout))
			resGets, err := h.MessageList(ctx, 100, "", map[message.Field]any{
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

func Test_MessageAssistantReplyExists(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC); return &t }()
	deletedTime := func() *time.Time { t := time.Date(2026, 4, 29, 12, 30, 0, 0, time.UTC); return &t }()

	pcA := uuid.FromStringOrNil("a4a4a4a4-2c11-11f1-aaaa-111111111111")
	pcB := uuid.FromStringOrNil("b5b5b5b5-2c11-11f1-bbbb-222222222222")

	tests := []struct {
		name string

		seed []*message.Message

		query     uuid.UUID
		expectRes bool
	}{
		{
			name: "delivered_match",
			seed: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c0000001-2c11-11f1-9000-000000000001"),
					},
					AIcallID:       uuid.FromStringOrNil("d0000001-2c11-11f1-9000-000000000001"),
					PipecatcallID:  pcA,
					DeliveryStatus: message.DeliveryStatusDelivered,
					Direction:      message.DirectionIncoming,
					Role:           message.RoleAssistant,
				},
			},
			query:     pcA,
			expectRes: true,
		},
		{
			name: "pending_only",
			seed: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c0000002-2c11-11f1-9000-000000000002"),
					},
					AIcallID:       uuid.FromStringOrNil("d0000002-2c11-11f1-9000-000000000002"),
					PipecatcallID:  pcA,
					DeliveryStatus: message.DeliveryStatusPending,
					Direction:      message.DirectionIncoming,
					Role:           message.RoleAssistant,
				},
			},
			query:     pcA,
			expectRes: false,
		},
		{
			name: "different_pcc",
			seed: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c0000003-2c11-11f1-9000-000000000003"),
					},
					AIcallID:       uuid.FromStringOrNil("d0000003-2c11-11f1-9000-000000000003"),
					PipecatcallID:  pcB,
					DeliveryStatus: message.DeliveryStatusDelivered,
					Direction:      message.DirectionIncoming,
					Role:           message.RoleAssistant,
				},
			},
			query:     pcA,
			expectRes: false,
		},
		{
			name: "wrong_role",
			seed: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c0000004-2c11-11f1-9000-000000000004"),
					},
					AIcallID:       uuid.FromStringOrNil("d0000004-2c11-11f1-9000-000000000004"),
					PipecatcallID:  pcA,
					DeliveryStatus: message.DeliveryStatusDelivered,
					Direction:      message.DirectionIncoming,
					Role:           message.RoleUser,
				},
			},
			query:     pcA,
			expectRes: false,
		},
		{
			name: "wrong_direction",
			seed: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c0000005-2c11-11f1-9000-000000000005"),
					},
					AIcallID:       uuid.FromStringOrNil("d0000005-2c11-11f1-9000-000000000005"),
					PipecatcallID:  pcA,
					DeliveryStatus: message.DeliveryStatusDelivered,
					Direction:      message.DirectionOutgoing,
					Role:           message.RoleAssistant,
				},
			},
			query:     pcA,
			expectRes: false,
		},
		{
			name: "deleted_row",
			seed: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c0000006-2c11-11f1-9000-000000000006"),
					},
					AIcallID:       uuid.FromStringOrNil("d0000006-2c11-11f1-9000-000000000006"),
					PipecatcallID:  pcA,
					DeliveryStatus: message.DeliveryStatusDelivered,
					Direction:      message.DirectionIncoming,
					Role:           message.RoleAssistant,
				},
			},
			query:     pcA,
			expectRes: false,
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

			// Seed messages via MessageCreate; cache calls are mocked.
			for _, m := range tt.seed {
				mockUtil.EXPECT().TimeNow().Return(curTime)
				mockCache.EXPECT().MessageSet(ctx, gomock.Any())
				if err := h.MessageCreate(ctx, m); err != nil {
					t.Fatalf("seed MessageCreate failed. err: %v", err)
				}
			}

			// For deleted_row: soft-delete the row after creating it.
			if tt.name == "deleted_row" {
				for _, m := range tt.seed {
					mockUtil.EXPECT().TimeNow().Return(deletedTime)
					mockCache.EXPECT().MessageSet(ctx, gomock.Any())
					if err := h.MessageDelete(ctx, m.ID); err != nil {
						t.Fatalf("seed MessageDelete failed. err: %v", err)
					}
				}
			}

			got, err := h.MessageAssistantReplyExists(ctx, tt.query)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if got != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, got)
			}

			// Cleanup so each subtest's seed doesn't bleed into the next via the shared in-memory DB.
			for _, m := range tt.seed {
				if _, err := dbTest.Exec("DELETE FROM ai_messages WHERE id = ?", m.ID.Bytes()); err != nil {
					t.Fatalf("cleanup failed. err: %v", err)
				}
			}
		})
	}
}

func Test_MessageUpdateDeliveryStatus(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC); return &t }()

	tests := []struct {
		name string

		seed *message.Message

		updateID     uuid.UUID
		updateStatus message.DeliveryStatus

		expectStatus message.DeliveryStatus
	}{
		{
			name: "pending_to_delivered",

			seed: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("e1000001-2c11-11f1-9000-000000000001"),
					CustomerID: uuid.FromStringOrNil("e1000001-2c11-11f1-9000-000000000002"),
				},
				AIcallID:       uuid.FromStringOrNil("e1000001-2c11-11f1-9000-000000000003"),
				PipecatcallID:  uuid.FromStringOrNil("e1000001-2c11-11f1-9000-000000000004"),
				DeliveryStatus: message.DeliveryStatusPending,
				Direction:      message.DirectionIncoming,
				Role:           message.RoleAssistant,
				Content:        "queued reply",
			},

			updateID:     uuid.FromStringOrNil("e1000001-2c11-11f1-9000-000000000001"),
			updateStatus: message.DeliveryStatusDelivered,

			expectStatus: message.DeliveryStatusDelivered,
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

			// Seed the message via MessageCreate.
			mockUtil.EXPECT().TimeNow().Return(curTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageCreate(ctx, tt.seed); err != nil {
				t.Fatalf("seed MessageCreate failed. err: %v", err)
			}

			// Update the delivery status. The implementation also refreshes the cache.
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageUpdateDeliveryStatus(ctx, tt.updateID, tt.updateStatus); err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Read back via MessageGet (cache miss → DB) and assert delivery_status was updated.
			mockCache.EXPECT().MessageGet(ctx, tt.updateID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			res, err := h.MessageGet(ctx, tt.updateID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.DeliveryStatus != tt.expectStatus {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectStatus, res.DeliveryStatus)
			}

			// Cleanup so this subtest's seed doesn't bleed into other tests via the shared in-memory DB.
			if _, err := dbTest.Exec("DELETE FROM ai_messages WHERE id = ?", tt.seed.ID.Bytes()); err != nil {
				t.Fatalf("cleanup failed. err: %v", err)
			}
		})
	}
}
