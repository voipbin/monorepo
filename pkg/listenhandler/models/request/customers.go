package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// V1DataCustomersPost is
// v1 data type request struct for
// /v1/customers POST
type V1DataCustomersPost struct {
	Username string `json:"username"`
	Password string `json:"password"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	WebhookMethod customer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                 `json:"webhook_uri,omitempty"`

	PermissionIDs []uuid.UUID `json:"permission_ids"`
}

// V1DataCustomersUsernameLoginPost is
// v1 data type request struct for
// /v1/customers/<username>/login POST
type V1DataCustomersUsernameLoginPost struct {
	Password string `json:"password"`
}

// V1DataCustomersIDPut is
// v1 data type request struct for
// /v1/customers/<customer-id> PUT
type V1DataCustomersIDPut struct {
	Name          string                 `json:"name,omitempty"`
	Detail        string                 `json:"detail,omitempty"`
	WebhookMethod customer.WebhookMethod `json:"webhook_method,omitempty"`
	WebhookURI    string                 `json:"webhook_uri,omitempty"`
}

// V1DataCustomersIDPasswordPut is
// v1 data type request struct for
// /v1/customers/<customer-id>/password PUT
type V1DataCustomersIDPasswordPut struct {
	Password string `json:"password"`
}

// V1DataCustomersIDPermissionIDsPut is
// v1 data type request struct for
// /v1/customers/<customer-id>/permission_ids PUT
type V1DataCustomersIDPermissionIDsPut struct {
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}
