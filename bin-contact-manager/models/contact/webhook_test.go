package contact

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
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
		Notes:       "Key enterprise customer - VIP account.",
		Addresses: []Address{
			{
				ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				Address: commonaddress.Address{
					Type:   AddressTypeTel,
					Target: "+155****4567",
				},
				IsPrimary: true,
			},
			{
				ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				Address: commonaddress.Address{
					Type:   AddressTypeEmail,
					Target: "john@example.com",
				},
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
	if webhook.Notes != contact.Notes {
		t.Errorf("Notes mismatch: got %v, want %v", webhook.Notes, contact.Notes)
	}
	if len(webhook.Addresses) != len(contact.Addresses) {
		t.Errorf("Addresses length mismatch: got %v, want %v", len(webhook.Addresses), len(contact.Addresses))
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

// TestAddress_JSON_EmbeddedFlattening proves that Address's embedded
// commonaddress.Address marshals its fields (type, target, ...) directly
// into the parent object rather than nesting them under an "address" key.
// Go's encoding/json flattens anonymous (embedded) struct fields into the
// enclosing object by default; this test locks that behavior in with a
// literal string assertion so a future accidental switch from embedding
// to a named field (e.g. `Address commonaddress.Address \`json:"address"\`)
// would be caught immediately.
func TestAddress_JSON_EmbeddedFlattening(t *testing.T) {
	addr := Address{
		ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		Address: commonaddress.Address{
			Type:   AddressTypeTel,
			Target: "+155****4567",
		},
		IsPrimary: true,
	}

	data, err := json.Marshal(addr)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	got := string(data)

	if strings.Contains(got, `"address":{`) {
		t.Errorf("JSON output contains a nested \"address\":{ key, embedded commonaddress.Address fields were not flattened. got: %s", got)
	}

	// Positive assertion: the embedded fields must appear flattened at
	// the top level of the JSON object.
	if !strings.Contains(got, `"type":"tel"`) {
		t.Errorf("JSON output missing flattened \"type\" field. got: %s", got)
	}
	if !strings.Contains(got, `"target":"+155****4567"`) {
		t.Errorf("JSON output missing flattened \"target\" field. got: %s", got)
	}
}

// TestContact_JSON_AddressEmbeddedFlattening runs the same flattening
// assertion at the Contact level (an Address nested inside Contact.Addresses)
// to confirm the embedding holds through the full webhook payload, not just
// a bare Address value.
func TestContact_JSON_AddressEmbeddedFlattening(t *testing.T) {
	c := &Contact{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
		},
		Addresses: []Address{
			{
				ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				Address: commonaddress.Address{
					Type:   AddressTypeEmail,
					Target: "john@example.com",
				},
			},
		},
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	got := string(data)

	if strings.Contains(got, `"address":{`) {
		t.Errorf("JSON output contains a nested \"address\":{ key, embedded commonaddress.Address fields were not flattened. got: %s", got)
	}
	if !strings.Contains(got, `"type":"email"`) {
		t.Errorf("JSON output missing flattened \"type\" field. got: %s", got)
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
			name: "contact with addresses",
			contact: &Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				},
				FirstName: "Jane",
				Addresses: []Address{
					{
						ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
						Address: commonaddress.Address{
							Type:   AddressTypeTel,
							Target: "+155****8888",
						},
					},
					{
						ID: uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
						Address: commonaddress.Address{
							Type:   AddressTypeEmail,
							Target: "jane@example.com",
						},
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
