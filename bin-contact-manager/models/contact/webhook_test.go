package contact

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestContact_ConvertWebhookMessage(t *testing.T) {
	contact := &Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		FirstName:   "John",
		LastName:    "Doe",
		DisplayName: "John Doe",
		Company:     "Acme Corp",
		JobTitle:    "Engineer",
		Source:      "manual",
		ExternalID:  "ext-123",
		PhoneNumbers: []PhoneNumber{
			{
				ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				Number:     "+1-555-123-4567",
				NumberE164: "+15551234567",
				Type:       "mobile",
				IsPrimary:  true,
			},
		},
		Emails: []Email{
			{
				ID:        uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				Address:   "john@example.com",
				Type:      "work",
				IsPrimary: true,
			},
		},
		TagIDs: []uuid.UUID{
			uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		},
		TMCreate: func() *time.Time { t := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
		TMUpdate: func() *time.Time { t := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC); return &t }(),
		TMDelete: nil,
	}

	webhook := contact.ConvertWebhookMessage()

	if webhook.ID != contact.ID {
		t.Errorf("ID mismatch: got %v, want %v", webhook.ID, contact.ID)
	}
	if webhook.CustomerID != contact.CustomerID {
		t.Errorf("CustomerID mismatch: got %v, want %v", webhook.CustomerID, contact.CustomerID)
	}
	if webhook.FirstName != contact.FirstName {
		t.Errorf("FirstName mismatch: got %v, want %v", webhook.FirstName, contact.FirstName)
	}
	if webhook.LastName != contact.LastName {
		t.Errorf("LastName mismatch: got %v, want %v", webhook.LastName, contact.LastName)
	}
	if webhook.DisplayName != contact.DisplayName {
		t.Errorf("DisplayName mismatch: got %v, want %v", webhook.DisplayName, contact.DisplayName)
	}
	if webhook.Company != contact.Company {
		t.Errorf("Company mismatch: got %v, want %v", webhook.Company, contact.Company)
	}
	if webhook.JobTitle != contact.JobTitle {
		t.Errorf("JobTitle mismatch: got %v, want %v", webhook.JobTitle, contact.JobTitle)
	}
	if webhook.Source != contact.Source {
		t.Errorf("Source mismatch: got %v, want %v", webhook.Source, contact.Source)
	}
	if webhook.ExternalID != contact.ExternalID {
		t.Errorf("ExternalID mismatch: got %v, want %v", webhook.ExternalID, contact.ExternalID)
	}
	if len(webhook.PhoneNumbers) != len(contact.PhoneNumbers) {
		t.Errorf("PhoneNumbers length mismatch: got %v, want %v", len(webhook.PhoneNumbers), len(contact.PhoneNumbers))
	}
	if len(webhook.Emails) != len(contact.Emails) {
		t.Errorf("Emails length mismatch: got %v, want %v", len(webhook.Emails), len(contact.Emails))
	}
	if len(webhook.TagIDs) != len(contact.TagIDs) {
		t.Errorf("TagIDs length mismatch: got %v, want %v", len(webhook.TagIDs), len(contact.TagIDs))
	}
	if (webhook.TMCreate == nil) != (contact.TMCreate == nil) || (webhook.TMCreate != nil && !webhook.TMCreate.Equal(*contact.TMCreate)) {
		t.Errorf("TMCreate mismatch: got %v, want %v", webhook.TMCreate, contact.TMCreate)
	}
	if (webhook.TMUpdate == nil) != (contact.TMUpdate == nil) || (webhook.TMUpdate != nil && !webhook.TMUpdate.Equal(*contact.TMUpdate)) {
		t.Errorf("TMUpdate mismatch: got %v, want %v", webhook.TMUpdate, contact.TMUpdate)
	}
	if (webhook.TMDelete == nil) != (contact.TMDelete == nil) {
		t.Errorf("TMDelete mismatch: got %v, want %v", webhook.TMDelete, contact.TMDelete)
	}
}

func TestContact_CreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name    string
		contact *Contact
		wantErr bool
	}{
		{
			name: "normal contact",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				TMCreate:    func() *time.Time { t := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
			},
			wantErr: false,
		},
		{
			name: "contact with phone numbers and emails",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName: "Jane",
				PhoneNumbers: []PhoneNumber{
					{
						ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
						Number:     "+1-555-999-8888",
						NumberE164: "+15559998888",
					},
				},
				Emails: []Email{
					{
						ID:      uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
						Address: "jane@example.com",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty contact",
			contact: &Contact{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.contact.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify it's valid JSON
				var webhook WebhookMessage
				if err := json.Unmarshal(data, &webhook); err != nil {
					t.Errorf("CreateWebhookEvent() returned invalid JSON: %v", err)
				}

				// Verify the ID matches
				if webhook.ID != tt.contact.ID {
					t.Errorf("Webhook ID mismatch: got %v, want %v", webhook.ID, tt.contact.ID)
				}
			}
		})
	}
}
