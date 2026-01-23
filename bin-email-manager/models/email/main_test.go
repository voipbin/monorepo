package email

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name string

		activeflowID        uuid.UUID
		providerType        ProviderType
		providerReferenceID string
		status              Status
		subject             string
		content             string
	}{
		{
			name: "creates_email_with_all_fields",

			activeflowID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			providerType:        ProviderTypeSendgrid,
			providerReferenceID: "sendgrid-ref-123",
			status:              StatusDelivered,
			subject:             "Test Subject",
			content:             "Test email content",
		},
		{
			name: "creates_email_with_empty_fields",

			activeflowID:        uuid.Nil,
			providerType:        "",
			providerReferenceID: "",
			status:              StatusNone,
			subject:             "",
			content:             "",
		},
		{
			name: "creates_email_with_initiated_status",

			activeflowID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			providerType:        ProviderTypeSendgrid,
			providerReferenceID: "sendgrid-ref-456",
			status:              StatusInitiated,
			subject:             "Welcome",
			content:             "Welcome to our service!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Email{
				ActiveflowID:        tt.activeflowID,
				ProviderType:        tt.providerType,
				ProviderReferenceID: tt.providerReferenceID,
				Status:              tt.status,
				Subject:             tt.subject,
				Content:             tt.content,
			}

			if e.ActiveflowID != tt.activeflowID {
				t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", tt.activeflowID, e.ActiveflowID)
			}
			if e.ProviderType != tt.providerType {
				t.Errorf("Wrong ProviderType. expect: %s, got: %s", tt.providerType, e.ProviderType)
			}
			if e.ProviderReferenceID != tt.providerReferenceID {
				t.Errorf("Wrong ProviderReferenceID. expect: %s, got: %s", tt.providerReferenceID, e.ProviderReferenceID)
			}
			if e.Status != tt.status {
				t.Errorf("Wrong Status. expect: %s, got: %s", tt.status, e.Status)
			}
			if e.Subject != tt.subject {
				t.Errorf("Wrong Subject. expect: %s, got: %s", tt.subject, e.Subject)
			}
			if e.Content != tt.content {
				t.Errorf("Wrong Content. expect: %s, got: %s", tt.content, e.Content)
			}
		})
	}
}

func TestProviderTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ProviderType
		expected string
	}{
		{
			name:     "provider_type_sendgrid",
			constant: ProviderTypeSendgrid,
			expected: "sendgrid",
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

func TestAttachment(t *testing.T) {
	tests := []struct {
		name string

		referenceType AttachmentReferenceType
		referenceID   uuid.UUID
	}{
		{
			name: "creates_attachment_with_recording",

			referenceType: AttachmentReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		},
		{
			name: "creates_attachment_with_none_type",

			referenceType: AttachmentReferenceTypeNone,
			referenceID:   uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Attachment{
				ReferenceType: tt.referenceType,
				ReferenceID:   tt.referenceID,
			}

			if a.ReferenceType != tt.referenceType {
				t.Errorf("Wrong ReferenceType. expect: %s, got: %s", tt.referenceType, a.ReferenceType)
			}
			if a.ReferenceID != tt.referenceID {
				t.Errorf("Wrong ReferenceID. expect: %s, got: %s", tt.referenceID, a.ReferenceID)
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
			name:     "status_none",
			constant: StatusNone,
			expected: "",
		},
		{
			name:     "status_initiated",
			constant: StatusInitiated,
			expected: "initiated",
		},
		{
			name:     "status_processed",
			constant: StatusProcessed,
			expected: "processed",
		},
		{
			name:     "status_delivered",
			constant: StatusDelivered,
			expected: "delivered",
		},
		{
			name:     "status_open",
			constant: StatusOpen,
			expected: "open",
		},
		{
			name:     "status_click",
			constant: StatusClick,
			expected: "click",
		},
		{
			name:     "status_bounce",
			constant: StatusBounce,
			expected: "bounce",
		},
		{
			name:     "status_dropped",
			constant: StatusDropped,
			expected: "dropped",
		},
		{
			name:     "status_deferred",
			constant: StatusDeffered,
			expected: "deferred",
		},
		{
			name:     "status_unsubscribe",
			constant: StatusUnsubscribe,
			expected: "unsubscribe",
		},
		{
			name:     "status_spamreport",
			constant: StatusSpamreport,
			expected: "spamreport",
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

func TestAttachmentReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant AttachmentReferenceType
		expected string
	}{
		{
			name:     "attachment_reference_type_none",
			constant: AttachmentReferenceTypeNone,
			expected: "",
		},
		{
			name:     "attachment_reference_type_recording",
			constant: AttachmentReferenceTypeRecording,
			expected: "recording",
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
