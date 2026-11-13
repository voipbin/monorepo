// Package configkeys declares the configuration keys of bin-billing-manager.
//
// The two keys are the Paddle credentials of the service: the API key that opens a customer
// portal session, and the secret that signs the webhooks of Paddle. They are independent secret
// keys. Each one is read at the time of use: a stored value wins, and the environment variable
// of the key is the fallback while no value is stored (see billingconfig).
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

	// KeyPaddleWebhookSecretKey is the secret that signs the webhooks of Paddle. The signature of
	// a webhook is checked only while a value exists.
	KeyPaddleWebhookSecretKey = "paddle.webhook_secret_key"
)

// maxSecretLen is the longest accepted Paddle credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations. A key without a stored row is read from the environment variable
// that is named here.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyPaddleAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "PADDLE_API_KEY", Description: "API key for the calls to the Paddle API (customer portal session). A stored value wins over the environment variable.", MaxLen: maxSecretLen},
		{Name: KeyPaddleWebhookSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "PADDLE_WEBHOOK_SECRET_KEY", Description: "Secret that signs the Paddle webhooks. The signature is checked only while a value exists. A stored value must equal the value of the hook service.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
