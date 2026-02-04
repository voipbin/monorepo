package message

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_MediaMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		media Media
	}{
		{
			name: "media_type_file",
			media: Media{
				Type:   MediaTypeFile,
				FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},
		},
		{
			name: "media_type_link",
			media: Media{
				Type:    MediaTypeLink,
				LinkURL: "https://example.com/image.png",
			},
		},
		{
			name: "media_type_address",
			media: Media{
				Type: MediaTypeAddress,
				Address: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15551234567",
				},
			},
		},
		{
			name: "media_type_agent",
			media: Media{
				Type: MediaTypeAgent,
				Agent: amagent.Agent{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b2c3d4e5-f6a7-8901-bcde-f23456789012"),
					},
					Username: "testagent",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.media)
			if err != nil {
				t.Errorf("Failed to marshal media: %v", err)
				return
			}

			// Unmarshal
			var result Media
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal media: %v", err)
				return
			}

			// Compare
			if !reflect.DeepEqual(tt.media, result) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.media, result)
			}
		})
	}
}

func Test_MediaArrayMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name   string
		medias []Media
	}{
		{
			name:   "empty_array",
			medias: []Media{},
		},
		{
			name: "single_file_media",
			medias: []Media{
				{
					Type:   MediaTypeFile,
					FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				},
			},
		},
		{
			name: "multiple_mixed_medias",
			medias: []Media{
				{
					Type:   MediaTypeFile,
					FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				},
				{
					Type:    MediaTypeLink,
					LinkURL: "https://example.com/document.pdf",
				},
				{
					Type: MediaTypeAddress,
					Address: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+15559876543",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.medias)
			if err != nil {
				t.Errorf("Failed to marshal medias: %v", err)
				return
			}

			// Unmarshal
			var result []Media
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal medias: %v", err)
				return
			}

			// Compare
			if !reflect.DeepEqual(tt.medias, result) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.medias, result)
			}
		})
	}
}

func Test_ConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		expect  *WebhookMessage
	}{
		{
			name: "message_with_empty_medias",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID:   uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:     TypeNormal,
				Text:     "Hello world",
				Medias:   []Media{},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
			expect: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID:   uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:     TypeNormal,
				Text:     "Hello world",
				Medias:   []Media{},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
		},
		{
			name: "message_with_file_media",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID: uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:   TypeNormal,
				Text:   "Check this file",
				Medias: []Media{
					{
						Type:   MediaTypeFile,
						FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					},
				},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
			expect: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID: uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:   TypeNormal,
				Text:   "Check this file",
				Medias: []Media{
					{
						Type:   MediaTypeFile,
						FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					},
				},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
		},
		{
			name: "message_with_link_media",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID: uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:   TypeNormal,
				Text:   "Check this link",
				Medias: []Media{
					{
						Type:    MediaTypeLink,
						LinkURL: "https://example.com/page",
					},
				},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
			expect: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID: uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:   TypeNormal,
				Text:   "Check this link",
				Medias: []Media{
					{
						Type:    MediaTypeLink,
						LinkURL: "https://example.com/page",
					},
				},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.message.ConvertWebhookMessage()
			if err != nil {
				t.Errorf("Failed to convert message: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expect, result)
			}
		})
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
	}{
		{
			name: "message_creates_valid_webhook_event",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID: uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:   TypeNormal,
				Text:   "Hello world",
				Medias: []Media{
					{
						Type:   MediaTypeFile,
						FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					},
				},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.message.CreateWebhookEvent()
			if err != nil {
				t.Errorf("Failed to create webhook event: %v", err)
				return
			}

			// Verify it's valid JSON by unmarshaling to WebhookMessage
			var wm WebhookMessage
			if err := json.Unmarshal(data, &wm); err != nil {
				t.Errorf("Webhook event is not valid JSON: %v", err)
				return
			}

			// Verify medias is an array, not a string
			if len(wm.Medias) != 1 {
				t.Errorf("Wrong medias count. expect: 1, got: %d", len(wm.Medias))
			}

			if wm.Medias[0].Type != MediaTypeFile {
				t.Errorf("Wrong media type. expect: %s, got: %s", MediaTypeFile, wm.Medias[0].Type)
			}
		})
	}
}

func Test_WebhookMessage_CreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name string
		wm   *WebhookMessage
	}{
		{
			name: "webhook_message_creates_valid_event",
			wm: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-a7b8-9012-cdef-345678901234"),
					CustomerID: uuid.FromStringOrNil("d4e5f6a7-b8c9-0123-def0-456789012345"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-1234-ef01-567890123456"),
				},
				ChatID: uuid.FromStringOrNil("f6a7b8c9-d0e1-2345-f012-678901234567"),
				Type:   TypeNormal,
				Text:   "Hello world",
				Medias: []Media{
					{
						Type:   MediaTypeFile,
						FileID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					},
				},
				Metadata: Metadata{Reactions: []Reaction{}},
				TMCreate: "2024-01-15T10:30:00.000000Z",
				TMUpdate: "2024-01-15T10:30:00.000000Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.wm.CreateWebhookEvent()
			if err != nil {
				t.Errorf("Failed to create webhook event: %v", err)
				return
			}

			// Verify it's valid JSON
			var result WebhookMessage
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Webhook event is not valid JSON: %v", err)
				return
			}

			// Verify data matches
			if !reflect.DeepEqual(&result, tt.wm) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.wm, &result)
			}
		})
	}
}
