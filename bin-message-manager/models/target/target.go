package target

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
)

// Target defines
type Target struct {
	Destination commonaddress.Address `json:"destination"`
	Status      Status                `json:"status"`
	Parts       int                   `json:"parts"` // segment count of the text computed by the platform from the standard, not the charged unit.
	TMUpdate    *time.Time            `json:"tm_update"`

	// internal keys. They are not part of the API (see WebhookTarget) and are cleared from the events.
	ProviderParts       int    `json:"provider_parts,omitempty"`        // segment count reported by the provider, for comparison only.
	ProviderReferenceID string `json:"provider_reference_id,omitempty"` // message id of the provider for this target.
}

// WebhookTarget is the part of a Target that is visible outside the platform.
type WebhookTarget struct {
	Destination commonaddress.Address `json:"destination"`
	Status      Status                `json:"status"`
	Parts       int                   `json:"parts"`
	TMUpdate    *time.Time            `json:"tm_update"`
}

// ConvertWebhookTarget converts to the WebhookTarget
func (h *Target) ConvertWebhookTarget() WebhookTarget {
	return WebhookTarget{
		Destination: h.Destination,
		Status:      h.Status,
		Parts:       h.Parts,
		TMUpdate:    h.TMUpdate,
	}
}

// ConvertWebhookTargets converts the targets to the WebhookTargets
func ConvertWebhookTargets(targets []Target) []WebhookTarget {
	if targets == nil {
		return nil
	}

	res := make([]WebhookTarget, 0, len(targets))
	for i := range targets {
		res = append(res, targets[i].ConvertWebhookTarget())
	}

	return res
}

// Status defines
type Status string

// list of Status defines
const (
	// inbound status
	StatusReceived Status = "received" // Received by the Telnyx Messaging Services.

	// outbound status
	StatusQueued     Status = "queued"      // Released from scheduler and submitted to gateway. [Queued in scheduler, due to outbound rate limiting.]
	StatusGWTimeout  Status = "gw_timeout"  // No DLR (delivery receipt) from gateway.
	StatusSent       Status = "sent"        // Success DLR received from gateway. Message has been sent downstream.
	StatusDLRTimeout Status = "dlr_timeout" // No DLR from downstream.
	StatusFailed     Status = "failed"      // Failure DLR from gateway or downstream, which is notification of message delivery failure.

	// both(inbound/outbound)
	StatusDelivered Status = "delivered" // outbound: To the best of our knowledge, the message was delivered./ inbound: Transmitted to you, after receipt.
)
