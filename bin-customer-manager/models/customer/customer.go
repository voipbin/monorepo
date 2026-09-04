package customer

import (
	"time"

	"github.com/gofrs/uuid"
)

// Status represents the account lifecycle state
type Status string

const (
	StatusInitial Status = "initial"
	StatusActive  Status = "active"
	StatusFrozen  Status = "frozen"
	StatusDeleted Status = "deleted"
	StatusExpired Status = "expired"
)

// Customer defines
type Customer struct {
	ID uuid.UUID `json:"id" db:"id,uuid"` // Customer's ID

	Name   string `json:"name,omitempty" db:"name"`     //  name
	Detail string `json:"detail,omitempty" db:"detail"` //  detail

	Email       string `json:"email,omitempty" db:"email"`
	PhoneNumber string `json:"phone_number,omitempty" db:"phone_number"`
	Address     string `json:"address,omitempty" db:"address"`

	// webhook info
	WebhookMethod WebhookMethod `json:"webhook_method,omitempty" db:"webhook_method"` // webhook method
	WebhookURI    string        `json:"webhook_uri,omitempty" db:"webhook_uri"`       // webhook uri
	WebhookSecret string        `json:"webhook_secret,omitempty" db:"webhook_secret"` // HMAC-SHA256 signing secret for outbound webhooks. Kept on the wire JSON (not json:"-") because bin-webhook-manager depends on it crossing the internal RPC/event-bus boundary (CustomerV1CustomerGet, customer_created/customer_updated events) -- that boundary is the trusted internal service layer, not the external API. External-facing responses MUST go through ConvertWebhookMessage(), which already excludes this field; the API gateway (bin-api-manager) is the security boundary, not this struct's json tags.

	BillingAccountID uuid.UUID `json:"billing_account_id,omitempty" db:"billing_account_id,uuid"` // default billing account id

	EmailVerified bool `json:"email_verified" db:"email_verified"`

	Status Status `json:"status" db:"status"`

	IdentityVerificationStatus IdentityVerificationStatus `json:"identity_verification_status" db:"identity_verification_status"`

	TermsAgreedVersion string `json:"terms_agreed_version,omitempty" db:"terms_agreed_version"`
	TermsAgreedIP      string `json:"terms_agreed_ip,omitempty" db:"terms_agreed_ip"`

	Metadata Metadata `json:"metadata" db:"metadata,json"` // internal options (admin-only)

	TMDeletionScheduled *time.Time `json:"tm_deletion_scheduled" db:"tm_deletion_scheduled"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"` // Deleted timestamp.
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *Customer) EventSubscriptionID() string {
	return h.ID.String()
}

// WebhookMethod defines
type WebhookMethod string

// list of methods
const (
	WebhookMethodNone   = ""
	WebhookMethodPost   = "POST"
	WebhookMethodGet    = "GET"
	WebhookMethodPut    = "PUT"
	WebhookMethodDelete = "DELETE"
)

var (
	IDEmpty = uuid.FromStringOrNil("00000000-0000-0000-0000-00000000000") //

	// voipbin internal service's customer id
	IDCallManager = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000001")
	IDAIManager   = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000002")

	// IDAIManagerListen is the customer id that owns transcribe sessions started
	// by the Insight AI's realtime call-listening feature (docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.2.1). It is deliberately
	// SEPARATE from IDAIManager -- bin-transcribe-manager's startLive duplicate
	// guard is scoped by customer_id, so sharing an owner with the AI-summary
	// feature would make the two collide on one session and entangle their
	// lifecycles. Like IDAIManager, this is a bare sentinel with no
	// customer_customers row behind it.
	//
	// NOTE on the literal: unlike the three sentinels above, this one is
	// well-formed (12 hex digits in the last group). The older literals are not,
	// so they all parse to uuid.Nil -- see TestSpecialIDConstants' own comment.
	// Copying their shape here would silently make this equal to IDAIManager.
	IDAIManagerListen = uuid.FromStringOrNil("00000000-0000-0000-0001-000000000003")

	// IDBasicRoute is the customer ID for system-wide default routes.
	// Used by route-manager to look up fallback routes when no customer-specific route exists.
	IDBasicRoute = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")

	// IDSystem is the customer ID used for system-generated operations
	// (e.g., signup verification emails, password reset emails).
	IDSystem = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000002")

	// GuestCustomerID is the guest/demo account customer id.
	GuestCustomerID = uuid.FromStringOrNil("a856c986-4b06-4496-9641-4d0ecbc67df5")
)
