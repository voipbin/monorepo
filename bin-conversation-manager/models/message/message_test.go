package message

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name string

		message *Message

		expectSource      commonaddress.Address
		expectDestination commonaddress.Address
	}{
		{
			name: "incoming sms carries source/destination",

			message: &Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
				},
				Direction: DirectionIncoming,
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+155****6543",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+155****0000",
				},
			},

			expectSource: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+155****6543",
			},
			expectDestination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+155****0000",
			},
		},
		{
			name: "outgoing message carries source/destination",

			message: &Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000002"),
				},
				Direction: DirectionOutgoing,
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+155****0000",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+155****6543",
				},
			},

			expectSource: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+155****0000",
			},
			expectDestination: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+155****6543",
			},
		},
		{
			name: "zero endpoints survive as zero address",

			message: &Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000003"),
				},
				Direction: DirectionNond,
			},

			expectSource:      commonaddress.Address{},
			expectDestination: commonaddress.Address{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.message.ConvertWebhookMessage()

			if !reflect.DeepEqual(res.Source, tt.expectSource) {
				t.Errorf("Wrong Source.\nexpect: %v\ngot: %v", tt.expectSource, res.Source)
			}
			if !reflect.DeepEqual(res.Destination, tt.expectDestination) {
				t.Errorf("Wrong Destination.\nexpect: %v\ngot: %v", tt.expectDestination, res.Destination)
			}
			if res.ID != tt.message.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.message.ID, res.ID)
			}
			if res.Direction != tt.message.Direction {
				t.Errorf("Wrong Direction. expect: %s, got: %s", tt.message.Direction, res.Direction)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name string

		conversationID uuid.UUID
		direction      Direction
		status         Status
		referenceType  ReferenceType
		referenceID    uuid.UUID
		transactionID  string
		text           string
	}{
		{
			name: "creates_message_with_all_fields",

			conversationID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			direction:      DirectionOutgoing,
			status:         StatusDone,
			referenceType:  ReferenceTypeMessage,
			referenceID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			transactionID:  "txn-123",
			text:           "Hello, world!",
		},
		{
			name: "creates_message_with_empty_fields",

			conversationID: uuid.Nil,
			direction:      DirectionNond,
			status:         "",
			referenceType:  ReferenceTypeNone,
			referenceID:    uuid.Nil,
			transactionID:  "",
			text:           "",
		},
		{
			name: "creates_incoming_line_message",

			conversationID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
			direction:      DirectionIncoming,
			status:         StatusProgressing,
			referenceType:  ReferenceTypeLine,
			referenceID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
			transactionID:  "line-txn-456",
			text:           "Message from LINE",
		},
		{
			name: "creates_failed_message",

			conversationID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
			direction:      DirectionOutgoing,
			status:         StatusFailed,
			referenceType:  ReferenceTypeMessage,
			referenceID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
			transactionID:  "txn-failed",
			text:           "Failed to send",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Message{
				ConversationID: tt.conversationID,
				Direction:      tt.direction,
				Status:         tt.status,
				ReferenceType:  tt.referenceType,
				ReferenceID:    tt.referenceID,
				TransactionID:  tt.transactionID,
				Text:           tt.text,
			}

			if m.ConversationID != tt.conversationID {
				t.Errorf("Wrong ConversationID. expect: %s, got: %s", tt.conversationID, m.ConversationID)
			}
			if m.Direction != tt.direction {
				t.Errorf("Wrong Direction. expect: %s, got: %s", tt.direction, m.Direction)
			}
			if m.Status != tt.status {
				t.Errorf("Wrong Status. expect: %s, got: %s", tt.status, m.Status)
			}
			if m.ReferenceType != tt.referenceType {
				t.Errorf("Wrong ReferenceType. expect: %s, got: %s", tt.referenceType, m.ReferenceType)
			}
			if m.ReferenceID != tt.referenceID {
				t.Errorf("Wrong ReferenceID. expect: %s, got: %s", tt.referenceID, m.ReferenceID)
			}
			if m.TransactionID != tt.transactionID {
				t.Errorf("Wrong TransactionID. expect: %s, got: %s", tt.transactionID, m.TransactionID)
			}
			if m.Text != tt.text {
				t.Errorf("Wrong Text. expect: %s, got: %s", tt.text, m.Text)
			}
		})
	}
}

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{
			name:     "field_id",
			constant: FieldID,
			expected: "id",
		},
		{
			name:     "field_customer_id",
			constant: FieldCustomerID,
			expected: "customer_id",
		},
		{
			name:     "field_conversation_id",
			constant: FieldConversationID,
			expected: "conversation_id",
		},
		{
			name:     "field_direction",
			constant: FieldDirection,
			expected: "direction",
		},
		{
			name:     "field_status",
			constant: FieldStatus,
			expected: "status",
		},
		{
			name:     "field_reference_type",
			constant: FieldReferenceType,
			expected: "reference_type",
		},
		{
			name:     "field_reference_id",
			constant: FieldReferenceID,
			expected: "reference_id",
		},
		{
			name:     "field_transaction_id",
			constant: FieldTransactionID,
			expected: "transaction_id",
		},
		{
			name:     "field_text",
			constant: FieldText,
			expected: "text",
		},
		{
			name:     "field_medias",
			constant: FieldMedias,
			expected: "medias",
		},
		{
			name:     "field_tm_create",
			constant: FieldTMCreate,
			expected: "tm_create",
		},
		{
			name:     "field_tm_update",
			constant: FieldTMUpdate,
			expected: "tm_update",
		},
		{
			name:     "field_tm_delete",
			constant: FieldTMDelete,
			expected: "tm_delete",
		},
		{
			name:     "field_deleted",
			constant: FieldDeleted,
			expected: "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{
			name:     "status_failed",
			constant: StatusFailed,
			expected: "failed",
		},
		{
			name:     "status_progressing",
			constant: StatusProgressing,
			expected: "progressing",
		},
		{
			name:     "status_done",
			constant: StatusDone,
			expected: "done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{
			name:     "direction_none",
			constant: DirectionNond,
			expected: "",
		},
		{
			name:     "direction_outgoing",
			constant: DirectionOutgoing,
			expected: "outgoing",
		},
		{
			name:     "direction_incoming",
			constant: DirectionIncoming,
			expected: "incoming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{
			name:     "reference_type_none",
			constant: ReferenceTypeNone,
			expected: "",
		},
		{
			name:     "reference_type_message",
			constant: ReferenceTypeMessage,
			expected: "message",
		},
		{
			name:     "reference_type_line",
			constant: ReferenceTypeLine,
			expected: "line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
