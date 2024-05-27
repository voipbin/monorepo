package request

import "github.com/gofrs/uuid"

// ParamStorageAccountsGET is request param define for
// GET /v1.0/storage_accounts
type ParamStorageAccountsGET struct {
	Pagination
}

// BodyStorageAccountsPOST is rquest body define for
// POST /v1.0/storage_accounts
type BodyStorageAccountsPOST struct {
	CustomerID uuid.UUID `json:"customer_id"`
}
