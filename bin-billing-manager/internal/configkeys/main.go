// Package configkeys declares the configuration keys of bin-billing-manager.
//
// The two keys are the Paddle credentials of the service: the API key that opens a customer
// portal session, and the secret that signs the webhooks of Paddle. They are independent secret
// keys. Each one is read from the database at the time of use. The keys have no environment
// variable and no default: a key without a stored value is not configured (see billingconfig).
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

// maxSecretLen is the longest accepted Paddle credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations. A key declares no environment variable, so a key without a
// stored row has no value.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyPaddleAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key for the calls to the Paddle API (customer portal session). Without a stored value a portal session can not be created.", MaxLen: maxSecretLen},
		{Name: KeyPaddleWebhookSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Secret that signs the Paddle webhooks. Without a stored value every Paddle webhook is rejected.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
