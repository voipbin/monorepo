package allowance

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Allowance represents a monthly token allowance cycle for a billing account.
type Allowance struct {
	commonidentity.Identity

	AccountID  uuid.UUID  `json:"account_id" db:"account_id,uuid"`
	CycleStart *time.Time `json:"cycle_start" db:"cycle_start"`
	CycleEnd   *time.Time `json:"cycle_end" db:"cycle_end"`

	TokensTotal int `json:"tokens_total" db:"tokens_total"`
	TokensUsed  int `json:"tokens_used" db:"tokens_used"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
