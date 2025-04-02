package account

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Account defines
type Account struct {
	commonidentity.Identity

	Type Type `json:"type"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Secret string `json:"secret"` // secret
	Token  string `json:"token"`  // usually api token

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type defines
type Type string

// list of types
const (
	TypeLine Type = "line"
	TypeSMS  Type = "sms"
)
