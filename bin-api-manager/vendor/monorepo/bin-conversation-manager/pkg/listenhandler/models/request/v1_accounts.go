package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/account"
)

// V1DataAccountsPost is
// v1 data type request struct for
// /v1/accounts POST
type V1DataAccountsPost struct {
	CustomerID uuid.UUID    `json:"customer_id"`
	Type       account.Type `json:"type"`
	Name       string       `json:"name"`
	Detail     string       `json:"detail"`
	Secret     string       `json:"secret"`
	Token      string       `json:"token"`
}

// V1DataAccountsIDPut is
// v1 data type request struct for
// /v1/accounts/<account-id> PUT
type V1DataAccountsIDPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Secret string `json:"secret"`
	Token  string `json:"token"`
}
