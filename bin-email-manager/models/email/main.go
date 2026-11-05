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

// list of ProviderType
const (
	ProviderTypeNone     ProviderType = "" // The provider is not decided yet (the email is not sent yet).
	ProviderTypeSendgrid ProviderType = "sendgrid"
	ProviderTypeMailgun  ProviderType = "mailgun"
)

type Attachment struct {
	ReferenceType AttachmentReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"`
}

// Status is the delivery status of the email.
//
// The values follow the SendGrid event names. The Mailgun webhook events map
// onto them as follows (see emailhandler/hook.go):
//
//	accepted                  -> processed
//	delivered                 -> delivered
//	failed (temporary)        -> deferred
//	failed (permanent)        -> bounce
//	opened                    -> open
//	clicked                   -> click
//	unsubscribed              -> unsubscribe
//	complained                -> spamreport
//
// The dropped status is produced by the SendGrid webhook only.
type Status string

const (
	StatusNone        Status = ""            // The email has no status.
	StatusInitiated   Status = "initiated"   // The email has been initiated.
	StatusProcessed   Status = "processed"   // The provider accepted the email and queued it for delivery.
	StatusDelivered   Status = "delivered"   // The email has been successfully delivered to the recipient's mail server.
	StatusOpen        Status = "open"        // The recipient opened the email.
	StatusClick       Status = "click"       // The recipient clicked on a link in the email.
	StatusBounce      Status = "bounce"      // The email could not be delivered (permanent failure).
	StatusDropped     Status = "dropped"     // The provider dropped the email (e.g., invalid recipient or suppression). SendGrid only.
	StatusDeffered    Status = "deferred"    // The provider has temporarily failed to deliver the email and will retry.
	StatusUnsubscribe Status = "unsubscribe" // The recipient unsubscribed from your email list.
	StatusSpamreport  Status = "spamreport"  // The recipient marked the email as spam.
)

type AttachmentReferenceType string

const (
	AttachmentReferenceTypeNone      AttachmentReferenceType = ""
	AttachmentReferenceTypeRecording AttachmentReferenceType = "recording"
)
