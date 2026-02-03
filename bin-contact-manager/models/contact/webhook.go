package contact

import (
	"encoding/json"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the payload sent in webhook notifications
// when contact events occur. This is a simplified version of Contact
// suitable for external consumption.
type WebhookMessage struct {
	commonidentity.Identity

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	Company     string `json:"company"`
	JobTitle    string `json:"job_title"`

	Source     string `json:"source"`
	ExternalID string `json:"external_id"`

	PhoneNumbers []PhoneNumber `json:"phone_numbers,omitempty"`
	Emails       []Email       `json:"emails,omitempty"`
	TagIDs       []uuid.UUID   `json:"tag_ids,omitempty"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts a Contact to a WebhookMessage
func (c *Contact) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: c.Identity,

		FirstName:   c.FirstName,
		LastName:    c.LastName,
		DisplayName: c.DisplayName,
		Company:     c.Company,
		JobTitle:    c.JobTitle,

		Source:     c.Source,
		ExternalID: c.ExternalID,

		PhoneNumbers: c.PhoneNumbers,
		Emails:       c.Emails,
		TagIDs:       c.TagIDs,

		TMCreate: c.TMCreate,
		TMUpdate: c.TMUpdate,
		TMDelete: c.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent as JSON bytes
func (c *Contact) CreateWebhookEvent() ([]byte, error) {
	e := c.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
