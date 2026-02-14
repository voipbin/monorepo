package allowance

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	cycleStart := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		allowance *Allowance
		expect    *WebhookMessage
	}{
		{
			name: "normal with all fields",
			allowance: &Allowance{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1234-5678-9abc-def012345678"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-2345-6789-abcd-ef0123456789"),
				},
				AccountID:   uuid.FromStringOrNil("c3d4e5f6-3456-789a-bcde-f01234567890"),
				CycleStart:  &cycleStart,
				CycleEnd:    &cycleEnd,
				TokensTotal: 10000,
				TokensUsed:  3500,
				TMCreate:    &tmCreate,
				TMUpdate:    &tmUpdate,
				TMDelete:    nil,
			},
			expect: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1234-5678-9abc-def012345678"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-2345-6789-abcd-ef0123456789"),
				},
				AccountID:   uuid.FromStringOrNil("c3d4e5f6-3456-789a-bcde-f01234567890"),
				CycleStart:  &cycleStart,
				CycleEnd:    &cycleEnd,
				TokensTotal: 10000,
				TokensUsed:  3500,
				TMCreate:    &tmCreate,
				TMUpdate:    &tmUpdate,
				TMDelete:    nil,
			},
		},
		{
			name: "zero tokens used",
			allowance: &Allowance{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d4e5f6a7-4567-89ab-cdef-012345678901"),
				},
				AccountID:   uuid.FromStringOrNil("e5f6a7b8-5678-9abc-def0-123456789012"),
				CycleStart:  &cycleStart,
				CycleEnd:    &cycleEnd,
				TokensTotal: 1000,
				TokensUsed:  0,
				TMCreate:    &tmCreate,
				TMUpdate:    &tmCreate,
			},
			expect: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d4e5f6a7-4567-89ab-cdef-012345678901"),
				},
				AccountID:   uuid.FromStringOrNil("e5f6a7b8-5678-9abc-def0-123456789012"),
				CycleStart:  &cycleStart,
				CycleEnd:    &cycleEnd,
				TokensTotal: 1000,
				TokensUsed:  0,
				TMCreate:    &tmCreate,
				TMUpdate:    &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.allowance.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot:    %+v", tt.expect, res)
			}
		})
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	cycleStart := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		allowance *Allowance
	}{
		{
			name: "normal",
			allowance: &Allowance{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-1234-5678-9abc-def012345678"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-2345-6789-abcd-ef0123456789"),
				},
				AccountID:   uuid.FromStringOrNil("c3d4e5f6-3456-789a-bcde-f01234567890"),
				CycleStart:  &cycleStart,
				CycleEnd:    &cycleEnd,
				TokensTotal: 10000,
				TokensUsed:  5000,
				TMCreate:    &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.allowance.CreateWebhookEvent()
			if err != nil {
				t.Errorf("Unexpected error. err: %v", err)
				return
			}

			// verify the output is valid JSON and round-trips correctly
			var msg WebhookMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Errorf("Could not unmarshal webhook event. err: %v", err)
				return
			}

			if msg.ID != tt.allowance.ID {
				t.Errorf("ID mismatch. expect: %s, got: %s", tt.allowance.ID, msg.ID)
			}
			if msg.AccountID != tt.allowance.AccountID {
				t.Errorf("AccountID mismatch. expect: %s, got: %s", tt.allowance.AccountID, msg.AccountID)
			}
			if msg.TokensTotal != tt.allowance.TokensTotal {
				t.Errorf("TokensTotal mismatch. expect: %d, got: %d", tt.allowance.TokensTotal, msg.TokensTotal)
			}
			if msg.TokensUsed != tt.allowance.TokensUsed {
				t.Errorf("TokensUsed mismatch. expect: %d, got: %d", tt.allowance.TokensUsed, msg.TokensUsed)
			}
		})
	}
}
