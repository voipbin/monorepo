package billing

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Billing define
type Billing struct {
	commonidentity.Identity

	AccountID uuid.UUID `json:"account_id" db:"account_id,uuid"` // billing account

	Status Status `json:"status" db:"status"`

	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`

	CostPerUnit float32 `json:"cost_per_unit" db:"cost_per_unit"`
	CostTotal   float32 `json:"cost_total" db:"cost_total"`

	BillingUnitCount float32 `json:"billing_unit_count" db:"billing_unit_count"`

	TMBillingStart *time.Time `json:"tm_billing_start" db:"tm_billing_start"`
	TMBillingEnd   *time.Time `json:"tm_billing_end" db:"tm_billing_end"`

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone           ReferenceType = ""
	ReferenceTypeCall           ReferenceType = "call"
	ReferenceTypeCallExtension  ReferenceType = "call_extension"
	ReferenceTypeSMS            ReferenceType = "sms"
	ReferenceTypeNumber         ReferenceType = "number"
	ReferenceTypeNumberRenew    ReferenceType = "number_renew"
	ReferenceTypeCreditFreeTier ReferenceType = "credit_free_tier"
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
	DefaultCostPerUnitReferenceTypeCall          float32 = 0.020
	DefaultCostPerUnitReferenceTypeCallExtension float32 = 0
	DefaultCostPerUnitReferenceTypeSMS           float32 = 0.008
	DefaultCostPerUnitReferenceTypeNumber        float32 = 5
)
