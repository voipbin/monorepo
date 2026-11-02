package pin

import (
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/message"
)

func Test_TargetType_IsValid(t *testing.T) {
	tests := []struct {
		target TargetType
		expect bool
	}{
		{TargetTypeChat, true},
		{TargetTypeMessage, true},
		{TargetType(""), false},
		{TargetType("call"), false},
	}

	for _, tt := range tests {
		if got := tt.target.IsValid(); got != tt.expect {
			t.Errorf("Wrong match. target: %q, expect: %v, got: %v", tt.target, tt.expect, got)
		}
	}
}

func Test_ConvertWebhookMessage(t *testing.T) {
	tm := time.Date(2026, 11, 2, 9, 15, 30, 0, time.UTC)
	id := uuid.FromStringOrNil("6f1c1a52-8d3b-4c1e-9a5e-0b7f2d9a1c10")
	targetID := uuid.FromStringOrNil("0d4b7a3e-2f5c-4b8e-8a21-9c3e6f1d7a42")
	chatID := uuid.FromStringOrNil("9a2b5c7d-1e3f-4a6b-8c9d-0e1f2a3b4c5d")
	customerID := uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")
	ownerID := uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796")

	msg := &message.Message{
		Identity: commonidentity.Identity{ID: targetID, CustomerID: customerID},
		ChatID:   chatID,
		Text:     "hello",
	}

	tests := []struct {
		name   string
		pin    *Pin
		expect *WebhookMessage
	}{
		{
			name: "chat pin hides the owner and the customer",
			pin: &Pin{
				Identity:   commonidentity.Identity{ID: id, CustomerID: customerID},
				Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
				ChatID:     chatID,
				TargetType: TargetTypeChat,
				TargetID:   chatID,
				TMCreate:   &tm,
			},
			expect: &WebhookMessage{ID: id, TargetType: TargetTypeChat, TargetID: chatID, ChatID: chatID, TMCreate: &tm},
		},
		{
			name: "message pin carries the converted message",
			pin: &Pin{
				Identity:   commonidentity.Identity{ID: id, CustomerID: customerID},
				Owner:      commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
				ChatID:     chatID,
				TargetType: TargetTypeMessage,
				TargetID:   targetID,
				TMCreate:   &tm,
				Message:    msg,
			},
			expect: &WebhookMessage{
				ID: id, TargetType: TargetTypeMessage, TargetID: targetID, ChatID: chatID, TMCreate: &tm,
				Message: &message.WebhookMessage{
					Identity: msg.Identity, ChatID: chatID, Text: "hello",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.pin.ConvertWebhookMessage()
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, res)
			}
		})
	}
}
