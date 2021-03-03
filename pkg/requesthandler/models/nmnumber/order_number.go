package nmnumber

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// Number struct represent number information
type Number struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
	FlowID uuid.UUID `json:"flow_id"`
	UserID uint64    `json:"user_id"`

	ProviderName        string `json:"provider_name"`
	ProviderReferenceID string `json:"provider_reference_id"`

	Status NumberStatus `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	// timestamp
	TMPurchase string `json:"tm_purchase"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// NumberStatus type
type NumberStatus string

// List of NumberStatus types
const (
	NumberStatusActive  NumberStatus = "active"
	NumberStatusDeleted NumberStatus = "deleted"
)

// ConvertNumber returns converted data from nmnumber.Number
func (t *Number) ConvertNumber() *models.Number {

	res := &models.Number{
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
