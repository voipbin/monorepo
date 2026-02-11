package extensiondirect

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for ExtensionDirect queries
type FieldStruct struct {
	ID          uuid.UUID `filter:"id"`
	CustomerID  uuid.UUID `filter:"customer_id"`
	ExtensionID uuid.UUID `filter:"extension_id"`
	Hash        string    `filter:"hash"`
	Deleted     bool      `filter:"deleted"`
}
