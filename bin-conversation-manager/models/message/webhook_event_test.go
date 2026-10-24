package message

import (
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
)

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tmUpdate := time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC)
	tmDelete := time.Date(2026, 1, 2, 3, 4, 7, 0, time.UTC)
	caseID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000009")

	m := &Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("33333333-0000-0000-0000-000000000002"),
		},
		ConversationID: uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003"),
		Direction:      DirectionIncoming,
		Status:         StatusDone,
		ReferenceType:  ReferenceTypeLine,
		ReferenceID:    uuid.FromStringOrNil("33333333-0000-0000-0000-000000000004"),
		Source:         commonaddress.Address{Type: commonaddress.TypeLine, Target: "peer-target"},
		Destination:    commonaddress.Address{Type: commonaddress.TypeLine, Target: "self-target"},
		Text:           "hello",
		Medias: []media.Media{
			{Type: media.TypeImage, Filename: "photo.png"},
		},
		TMCreate: &tmCreate,
		TMUpdate: &tmUpdate,
		TMDelete: &tmDelete,
	}
	// the internal case hint must never reach the customer-facing payload.
	m.CaseID = &caseID

	res, err := m.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expected := `{"id":"33333333-0000-0000-0000-000000000001","customer_id":"33333333-0000-0000-0000-000000000002","conversation_id":"33333333-0000-0000-0000-000000000003","direction":"incoming","status":"done","reference_type":"line","reference_id":"33333333-0000-0000-0000-000000000004","source":{"type":"line","target":"peer-target"},"destination":{"type":"line","target":"self-target"},"text":"hello","medias":[{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","type":"image","filename":"photo.png","tm_create":null,"tm_update":null,"tm_delete":null}],"tm_create":"2026-01-02T03:04:05Z","tm_update":"2026-01-02T03:04:06Z","tm_delete":"2026-01-02T03:04:07Z"}`
	if string(res) != expected {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expected, res)
	}
}

func Test_ConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name string
		src  map[string]any

		expectRes map[Field]any
		expectErr bool
	}{
		{
			name: "uuid strings are converted and plain strings pass through",
			src: map[string]any{
				"conversation_id": "44444444-0000-0000-0000-000000000001",
				"text":            "hello",
			},
			expectRes: map[Field]any{
				FieldConversationID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001"),
				FieldText:           "hello",
			},
		},
		{
			name: "invalid uuid string returns error",
			src: map[string]any{
				"conversation_id": "not-a-uuid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ConvertStringMapToFieldMap(tt.src)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				return
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
