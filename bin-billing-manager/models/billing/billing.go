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

	CostType          CostType `json:"cost_type" db:"cost_type"`
	CostUnitCount     float32  `json:"cost_unit_count" db:"cost_unit_count"`
	CostTokenPerUnit  int      `json:"cost_token_per_unit" db:"cost_token_per_unit"`
	CostTokenTotal    int      `json:"cost_token_total" db:"cost_token_total"`
	CostCreditPerUnit float32  `json:"cost_credit_per_unit" db:"cost_credit_per_unit"`
	CostCreditTotal   float32  `json:"cost_credit_total" db:"cost_credit_total"`

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

