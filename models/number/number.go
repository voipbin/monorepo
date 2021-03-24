package number

import (
	"github.com/gofrs/uuid"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// Number struct represent order number information
type Number struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
	FlowID uuid.UUID `json:"flow_id"`
	UserID uint64    `json:"-"` // we don't expose this to the client.

	Status string `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	// timestamp
	TMPurchase string `json:"tm_purchase"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertNumber returns converted data from nmnumber.Number
func ConvertNumber(t *nmnumber.Number) *Number {

	res := &Number{
		ID:     t.ID,
		Number: t.Number,
		FlowID: t.FlowID,
		UserID: t.UserID,

		Status:           string(t.Status),
		T38Enabled:       t.T38Enabled,
		EmergencyEnabled: t.EmergencyEnabled,
		TMPurchase:       t.TMPurchase,
		TMCreate:         t.TMCreate,
		TMUpdate:         t.TMUpdate,
		TMDelete:         t.TMDelete,
	}

	return res
}

// CreateNumber returns converted data from number.Number to nmnumber.Number
func CreateNumber(f *Number) *nmnumber.Number {

	res := &nmnumber.Number{
		ID:     f.ID,
		Number: f.Number,
		UserID: f.UserID,
		FlowID: f.FlowID,

		Status:           nmnumber.Status(f.Status),
		T38Enabled:       f.T38Enabled,
		EmergencyEnabled: f.EmergencyEnabled,
		TMPurchase:       f.TMPurchase,
		TMCreate:         f.TMCreate,
		TMUpdate:         f.TMUpdate,
		TMDelete:         f.TMDelete,
	}

	return res
}
