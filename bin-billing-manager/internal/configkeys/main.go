// Package configkeys declares the configuration keys of bin-billing-manager.
//
// The two keys are the Paddle credentials of the service: the API key that opens a customer
// portal session, and the secret that signs the webhooks of Paddle. They are independent secret
// keys. The service reads each one from its environment variable (the Env... constants) at the
// time of use (see billingconfig). The keys declare no Env and no default: the declarations serve
// the stored settings (admin API), whose values are not read for a use.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "billing_configs"
	HistoryTable = "billing_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To change the kind of a key, declare a new key name.
const (
	// KeyPaddleAPIKey is the API key that authenticates the calls to the Paddle API (the customer
	// portal session).
	KeyPaddleAPIKey = "paddle.api_key"

	// KeyPaddleWebhookSecretKey is the secret that signs the webhooks of Paddle. A webhook is
	// rejected while no value is stored.
	KeyPaddleWebhookSecretKey = "paddle.webhook_secret_key"
)

// Names of the environment variables that the service reads. A variable that is unset, empty or an
// unresolved placeholder means that the key is not configured.
const (
	// EnvPaddleAPIKey is the variable of the Paddle API key.
	EnvPaddleAPIKey = "PADDLE_API_KEY"

	// EnvPaddleWebhookSecret is the variable of the secret that signs the Paddle webhooks.
	EnvPaddleWebhookSecret = "PADDLE_WEBHOOK_SECRET"
)

// EnvName returns the variable that the service reads for the key.
func EnvName(key string) (string, bool) {
	switch key {
	case KeyPaddleAPIKey:
		return EnvPaddleAPIKey, true
	case KeyPaddleWebhookSecretKey:
		return EnvPaddleWebhookSecret, true
	default:
		return "", false
	}
}

// maxSecretLen is the longest accepted Paddle credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations of the stored settings. A key declares no environment variable.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyPaddleAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key for the calls to the Paddle API (customer portal session). The stored value is not used: the service reads PADDLE_API_KEY.", MaxLen: maxSecretLen},
		{Name: KeyPaddleWebhookSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Secret that signs the Paddle webhooks. The stored value is not used: the service reads PADDLE_WEBHOOK_SECRET.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
