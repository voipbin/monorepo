// Package configkeys declares the configuration keys of bin-message-manager.
//
// The two keys are the authentication tokens of the SMS and MMS providers. They are stored in the
// database only. They are independent secret keys: each provider has its own token and they
// are rotated separately. The Telnyx token of this service is a different setting from the Telnyx
// token of bin-number-manager (a different source), so rotating one
// does not rotate the other.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "message_configs"
	HistoryTable = "message_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To make a key secret, declare a new key name.
const (
	// KeyMessagebirdAuthToken is the access key of the MessageBird account that sends the SMS and
	// MMS that Telnyx refused.
	KeyMessagebirdAuthToken = "provider.messagebird.auth_token"

	// KeyTelnyxAuthToken is the API key of the Telnyx account that sends the SMS and MMS of this
	// service. It is not the token of bin-number-manager.
	KeyTelnyxAuthToken = "provider.telnyx.auth_token"
)

// maxSecretLen is the longest accepted provider token in bytes.
const maxSecretLen = 512

// Keys returns the declarations. The values are stored in the database only: a key has no
// environment name, so a key without a row is not configured.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyMessagebirdAuthToken, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Access key of the MessageBird account that sends the SMS and MMS that Telnyx refused.", MaxLen: maxSecretLen},
		{Name: KeyTelnyxAuthToken, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the Telnyx account that sends the SMS and MMS of the message service. It is read from here and not from the number service, so a key rotation needs both places.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
