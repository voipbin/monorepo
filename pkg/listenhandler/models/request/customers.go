package request

import "github.com/gofrs/uuid"

// V1DataCustomersPost is
// v1 data type request struct for
// /v1/customers POST
type V1DataCustomersPost struct {
	Username string `json:"username"`
	Password string `json:"password"`

	Name          string `json:"name"`
	Detail        string `json:"detail"`
	WebhookMethod string `json:"webhook_method"`
	WebhookURI    string `json:"webhook_uri"`

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
	Name          string `json:"name"`
	Detail        string `json:"detail"`
	WebhookMethod string `json:"webhook_method"`
	WebhookURI    string `json:"webhook_uri"`
}

// V1DataUsersIDPasswordPut is
// v1 data type request struct for
// /v1/customers/<customer-id>/password PUT
type V1DataUsersIDPasswordPut struct {
	Password string `json:"password"`
}

// V1DataCustomersIDPermissionPut is
// v1 data type request struct for
// /v1/customers/<customer-id>/permission PUT
type V1DataCustomersIDPermissionPut struct {
	PermissionIDs []uuid.UUID `json:"permission_ids"`
}
