package billing

import (
	"github.com/gofrs/uuid"
)

// Billing define
type Billing struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	AccountID  uuid.UUID `json:"account_id"` // billing account

	Status Status `json:"status"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	CostPerUnit float32 `json:"cost_per_unit"`
	CostTotal   float32 `json:"cost_total"`

	BillingUnitCount float32 `json:"billing_unit_count"`

	TMBillingStart string `json:"tm_billing_start"`
	TMBillingEnd   string `json:"tm_billing_end"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone        ReferenceType = ""
	ReferenceTypeCall        ReferenceType = "call"
	ReferenceTypeSMS         ReferenceType = "sms"
	ReferenceTypeNumber      ReferenceType = "number"
	ReferenceTypeNumberRenew ReferenceType = "number_renew"
)

// Status define
type Status string

// list of status
const (
	StatusProgressing Status = "progressing"
	StatusEnd         Status = "end"
	StatusPending     Status = "pending"
	StatusFinished    Status = "finished"
)

// list of default billing info
const (
	DefaultCostPerUnitReferenceTypeCall   float32 = 0.020
	DefaultCostPerUnitReferenceTypeSMS    float32 = 0.008
	DefaultCostPerUnitReferenceTypeNumber float32 = 5
)
