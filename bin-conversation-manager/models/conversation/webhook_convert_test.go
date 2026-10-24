package conversation

import (
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tmUpdate := time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC)
	tmDelete := time.Date(2026, 1, 2, 3, 4, 7, 0, time.UTC)

	cv := &Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003"),
		},
		AccountID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000004"),
		Name:      "test name",
		Detail:    "test detail",
		Type:      TypeLine,
		DialogID:  "dialog-1",
		Self:      commonaddress.Address{Type: commonaddress.TypeLine, Target: "self-target"},
		Peer:      commonaddress.Address{Type: commonaddress.TypeLine, Target: "peer-target", TargetName: "peer name"},
		Metadata:  &Metadata{},
		TMCreate:  &tmCreate,
		TMUpdate:  &tmUpdate,
		TMDelete:  &tmDelete,
	}

	res := cv.ConvertWebhookMessage()

	expected := &WebhookMessage{
		Identity:  cv.Identity,
		Owner:     cv.Owner,
		AccountID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000004"),
		Name:      "test name",
		Detail:    "test detail",
		Type:      TypeLine,
		DialogID:  "dialog-1",
		Self:      commonaddress.Address{Type: commonaddress.TypeLine, Target: "self-target"},
		Peer:      commonaddress.Address{Type: commonaddress.TypeLine, Target: "peer-target", TargetName: "peer name"},
		TMCreate:  &tmCreate,
		TMUpdate:  &tmUpdate,
		TMDelete:  &tmDelete,
	}
	if !reflect.DeepEqual(res, expected) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expected, res)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	cv := &Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003"),
		},
		AccountID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000004"),
		Name:      "test name",
		Detail:    "test detail",
		Type:      TypeLine,
		DialogID:  "dialog-1",
		Self:      commonaddress.Address{Type: commonaddress.TypeLine, Target: "self-target"},
		Peer:      commonaddress.Address{Type: commonaddress.TypeLine, Target: "peer-target"},
		Metadata:  &Metadata{},
		TMCreate:  &tmCreate,
	}

	res, err := cv.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// metadata is internal-only and must not appear in the webhook payload.
	expected := `{"id":"11111111-0000-0000-0000-000000000001","customer_id":"11111111-0000-0000-0000-000000000002","owner_type":"agent","owner_id":"11111111-0000-0000-0000-000000000003","account_id":"11111111-0000-0000-0000-000000000004","name":"test name","detail":"test detail","type":"line","dialog_id":"dialog-1","self":{"type":"line","target":"self-target"},"peer":{"type":"line","target":"peer-target"},"tm_create":"2026-01-02T03:04:05Z","tm_update":null,"tm_delete":null}`
	if string(res) != expected {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expected, res)
	}
}

func Test_ConvertStringMapToFieldMap(t *testing.T) {
	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		src  map[string]any

		expectRes map[Field]any
		expectErr bool
	}{
		{
			name: "uuid strings are converted",
			src: map[string]any{
				"id":          "22222222-0000-0000-0000-000000000001",
				"customer_id": "22222222-0000-0000-0000-000000000002",
			},
			expectRes: map[Field]any{
				FieldID:         uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001"),
				FieldCustomerID: uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
			},
		},
		{
			name: "same type values pass through, deleted string is converted to bool, unknown keys pass through",
			src: map[string]any{
				"tm_create": &tmCreate,
				"deleted":   "true",
				"unknown":   "keep",
				"dialog_id": "dialog-1",
			},
			expectRes: map[Field]any{
				FieldTMCreate:    &tmCreate,
				FieldDeleted:     true,
				Field("unknown"): "keep",
				FieldDialogID:    "dialog-1",
			},
		},
		{
			name: "invalid uuid string returns error",
			src: map[string]any{
				"customer_id": "not-a-uuid",
			},
			expectErr: true,
		},
		{
			name: "string for time field returns error",
			src: map[string]any{
				"tm_create": "2026-01-02T03:04:05Z",
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
