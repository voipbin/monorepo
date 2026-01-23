package chat

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestChatStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	c := Chat{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type:        TypeDirect,
		Name:        "Test Chat",
		Detail:      "Test chat description",
		MemberCount: 2,
		TMCreate:    "2023-01-01 00:00:00",
		TMUpdate:    "2023-01-02 00:00:00",
		TMDelete:    "",
	}

	if c.ID != id {
		t.Errorf("Chat.ID = %v, expected %v", c.ID, id)
	}
	if c.CustomerID != customerID {
		t.Errorf("Chat.CustomerID = %v, expected %v", c.CustomerID, customerID)
	}
	if c.Type != TypeDirect {
		t.Errorf("Chat.Type = %v, expected %v", c.Type, TypeDirect)
	}
	if c.Name != "Test Chat" {
		t.Errorf("Chat.Name = %v, expected %v", c.Name, "Test Chat")
	}
	if c.Detail != "Test chat description" {
		t.Errorf("Chat.Detail = %v, expected %v", c.Detail, "Test chat description")
	}
	if c.MemberCount != 2 {
		t.Errorf("Chat.MemberCount = %v, expected %v", c.MemberCount, 2)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_direct", TypeDirect, "direct"},
		{"type_group", TypeGroup, "group"},
		{"type_talk", TypeTalk, "talk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestChat_ConvertWebhookMessage(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	c := Chat{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type:        TypeGroup,
		Name:        "Team Chat",
		Detail:      "Team communication channel",
		MemberCount: 5,
		TMCreate:    "2023-01-01 00:00:00",
		TMUpdate:    "2023-01-02 00:00:00",
	}

	result := c.ConvertWebhookMessage()

	if result.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", result.ID, id)
	}
	if result.CustomerID != customerID {
		t.Errorf("WebhookMessage.CustomerID = %v, expected %v", result.CustomerID, customerID)
	}
	if result.Type != TypeGroup {
		t.Errorf("WebhookMessage.Type = %v, expected %v", result.Type, TypeGroup)
	}
	if result.Name != "Team Chat" {
		t.Errorf("WebhookMessage.Name = %v, expected %v", result.Name, "Team Chat")
	}
	if result.MemberCount != 5 {
		t.Errorf("WebhookMessage.MemberCount = %v, expected %v", result.MemberCount, 5)
	}
}

func TestChat_CreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	c := Chat{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type:        TypeDirect,
		Name:        "Test Chat",
		MemberCount: 2,
		TMCreate:    "2023-01-01 00:00:00",
	}

	data, err := c.CreateWebhookEvent()
	if err != nil {
		t.Errorf("CreateWebhookEvent failed: %v", err)
		return
	}

	var wm WebhookMessage
	if err := json.Unmarshal(data, &wm); err != nil {
		t.Errorf("Failed to unmarshal webhook event: %v", err)
		return
	}

	if wm.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", wm.ID, id)
	}
	if wm.Type != TypeDirect {
		t.Errorf("WebhookMessage.Type = %v, expected %v", wm.Type, TypeDirect)
	}
}

func TestGetDBFields(t *testing.T) {
	fields := GetDBFields()

	expectedFields := []string{
		"id",
		"customer_id",
		"type",
		"name",
		"detail",
		"member_count",
		"tm_create",
		"tm_update",
		"tm_delete",
	}

	if !reflect.DeepEqual(fields, expectedFields) {
		t.Errorf("GetDBFields() = %v, expected %v", fields, expectedFields)
	}
}
