package email

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Email defines
type Email struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id" db:"activeflow_id,uuid"`

	// provider info
	ProviderType        ProviderType `json:"provider_type" db:"provider_type"`
	ProviderReferenceID string       `json:"provider_reference_id" db:"provider_reference_id"`

	// from/to info
	Source       *commonaddress.Address  `json:"source" db:"source,json"`
	Destinations []commonaddress.Address `json:"destinations" db:"destinations,json"`

	// message info
	Status  Status `json:"status" db:"status"`
	Subject string `json:"subject" db:"subject"` // Subject of the message.
	Content string `json:"content" db:"content"` // Content of the message.

	Attachments []Attachment `json:"attachments" db:"attachments,json"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// ProviderType type
type ProviderType string

// list of NumberProvider
const (
	ProviderTypeSendgrid ProviderType = "sendgrid"
)

type Attachment struct {
	ReferenceType AttachmentReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"`
}

type Status string

const (
	StatusNone        Status = ""            // The email has no status.
	StatusInitiated   Status = "initiated"   // The email has been initiated.
	StatusProcessed   Status = "processed"   // The email has been received is being processed.
	StatusDelivered   Status = "delivered"   // The email has been successfully delivered to the recipient's inbox (or spam folder).
	StatusOpen        Status = "open"        // The recipient opened the email.
	StatusClick       Status = "click"       // The recipient clicked on a link in the email.
	StatusBounce      Status = "bounce"      // The email bounced (permanent or temporary failure). The status and reason fields provide more information.
	StatusDropped     Status = "dropped"     // SendGrid dropped the email (e.g., due to an invalid recipient, spam report, or blocked IP address). The reason field indicates the reason for the drop.
	StatusDeffered    Status = "deferred"    // SendGrid has temporarily deferred delivery of the email. They will attempt to deliver it later.
	StatusUnsubscribe Status = "unsubscribe" // The recipient unsubscribed from your email list.
	StatusSpamreport  Status = "spamreport"  // The recipient marked the email as spam.
)

type AttachmentReferenceType string

const (
	AttachmentReferenceTypeNone      AttachmentReferenceType = ""
	AttachmentReferenceTypeRecording AttachmentReferenceType = "recording"
)
