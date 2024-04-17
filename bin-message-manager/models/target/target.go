package target

import (
	commonaddress "monorepo/bin-common-handler/models/address"
)

// Target defines
type Target struct {
	Destination commonaddress.Address `json:"destination"`
	Status      Status                `json:"status"`
	Parts       int                   `json:"parts"` // number of messages
	TMUpdate    string                `json:"tm_update"`
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
