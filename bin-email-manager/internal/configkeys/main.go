// Package configkeys declares the configuration keys of bin-email-manager.
//
// The three keys are the provider credentials. The service reads them from environment variables
// (the Env constants below); the keys declare no environment name (Key.Env stays empty), so the
// admin listing does not suggest an environment source. They are independent keys on purpose: the Mailgun API key (sending) and the Mailgun webhook signing key
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

// Names of the environment variables that the service reads. A variable that is unset, empty or an
// unresolved placeholder means that the key is not configured.
const (
	EnvSendgridAPIKey           = "SENDGRID_API_KEY"
	EnvMailgunAPIKey            = "MAILGUN_API_KEY"
	EnvMailgunWebhookSigningKey = "MAILGUN_WEBHOOK_SIGNING_KEY"
)

// EnvName returns the variable that the service reads for the key.
func EnvName(key string) (string, bool) {
	switch key {
	case KeySendgridAPIKey:
		return EnvSendgridAPIKey, true
	case KeyMailgunAPIKey:
		return EnvMailgunAPIKey, true
	case KeyMailgunWebhookSigningKey:
		return EnvMailgunWebhookSigningKey, true
	default:
		return "", false
	}
}

// maxSecretLen is the longest accepted provider credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations of the stored settings. A key declares no environment name.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeySendgridAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the SendGrid account that sends email.", MaxLen: maxSecretLen},
		{Name: KeyMailgunAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the Mailgun account that sends email.", MaxLen: maxSecretLen},
		{Name: KeyMailgunWebhookSigningKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Key that verifies the signature of the Mailgun delivery webhooks.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
