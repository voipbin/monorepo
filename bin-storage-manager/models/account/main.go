package account

import "github.com/gofrs/uuid"

// Account defines
type Account struct {
	ID         uuid.UUID `json:"id" db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	TotalFileCount int64 `json:"total_file_count" db:"total_file_count"` // total file count
	TotalFileSize  int64 `json:"total_file_size" db:"total_file_size"`   // total file size in bytes

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}
