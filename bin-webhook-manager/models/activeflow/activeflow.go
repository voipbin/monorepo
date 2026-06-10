package activeflow

import (
	"time"

	"monorepo/bin-webhook-manager/models/webhook"
)

// Webhook represents the cached per-activeflow webhook destination.
//
// It is stored in Redis under the key `webhook:activeflow:{id}` as JSON and is
// used as both a positive entry (a real destination is configured) and a
// negative tombstone (no destination, deleted, or transient miss).
//
//   - Positive: Deleted == false && URI != ""
//   - Negative: Deleted == true (tombstone) OR URI == ""
//
// Tm carries the source event timestamp and is used to keep cache writes
// monotonic (a write only applies if its timestamp is not older than the stored
// one), guarding against the created-after-deleted resurrection race (design 5.6).
type Webhook struct {
	URI    string             `json:"uri,omitempty"`
	Method webhook.MethodType `json:"method,omitempty"`

	// Deleted marks a negative tombstone produced by an activeflow_deleted event
	// or a soft-deleted fallback result.
	Deleted bool `json:"deleted,omitempty"`

	// TMDelete carries the delete timestamp when this entry is a delete tombstone.
	TMDelete *time.Time `json:"tm_delete,omitempty"`

	// Tm is the source event timestamp used for monotonic ordering of writes.
	Tm time.Time `json:"tm"`
}

// IsPositive returns true when this entry represents a real, deliverable
// per-activeflow webhook destination.
func (w *Webhook) IsPositive() bool {
	return w != nil && !w.Deleted && w.URI != ""
}
