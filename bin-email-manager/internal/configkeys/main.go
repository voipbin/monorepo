// Package configkeys declares the configuration keys of bin-email-manager.
//
// The three keys are the provider credentials that were read from the environment only. They are
// independent keys on purpose: the Mailgun API key (sending) and the Mailgun webhook signing key
// (receiving) have different purposes and are rotated separately.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "email_configs"
	HistoryTable = "email_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To make a key secret, declare a new key name.
const (
	KeySendgridAPIKey           = "sendgrid.api_key"
	KeyMailgunAPIKey            = "mailgun.api_key"
	KeyMailgunWebhookSigningKey = "mailgun.webhook_signing_key"
)

// maxSecretLen is the longest accepted provider credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations. The environment names are the variables that the service read
// before the configs existed, so a database without rows behaves exactly as before.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeySendgridAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the SendGrid account that sends email.", Env: "SENDGRID_API_KEY", MaxLen: maxSecretLen},
		{Name: KeyMailgunAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the Mailgun account that sends email.", Env: "MAILGUN_API_KEY", MaxLen: maxSecretLen},
		{Name: KeyMailgunWebhookSigningKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Key that verifies the signature of the Mailgun delivery webhooks.", Env: "MAILGUN_WEBHOOK_SIGNING_KEY", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
