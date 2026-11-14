// Package configkeys declares the configuration keys of bin-message-manager.
//
// The two keys are the authentication tokens of the SMS and MMS providers. The service reads the
// tokens from the environment variables named below, and only from there; the stored value of a key
// is not read. They are independent secret keys: each provider has its own token and they are
// rotated separately. The Telnyx key is the same variable (TELNYX_API_KEY) that bin-number-manager
// reads, so one rotation covers both services.
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

// Names of the environment variables that the service reads.
const (
	// EnvMessagebirdAPIKey is the variable that holds the MessageBird access key.
	EnvMessagebirdAPIKey = "MESSAGEBIRD_API_KEY"

	// EnvTelnyxAPIKey is the variable that holds the Telnyx API key. It is shared with
	// bin-number-manager.
	EnvTelnyxAPIKey = "TELNYX_API_KEY"
)

// EnvName returns the variable that the service reads for the key, or an empty text for an unknown
// key.
func EnvName(key string) string {
	switch key {
	case KeyMessagebirdAuthToken:
		return EnvMessagebirdAPIKey
	case KeyTelnyxAuthToken:
		return EnvTelnyxAPIKey
	default:
		return ""
	}
}

// maxSecretLen is the longest accepted provider token in bytes.
const maxSecretLen = 512

// Keys returns the declarations of the stored settings. A key has no environment name (Env stays
// empty): the handler of the stored settings never reads the environment, and the service reads the
// variables above directly.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyMessagebirdAuthToken, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Access key of the MessageBird account that sends the SMS and MMS that Telnyx refused. The service reads MESSAGEBIRD_API_KEY; the stored value is not used.", MaxLen: maxSecretLen},
		{Name: KeyTelnyxAuthToken, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the Telnyx account that sends the SMS and MMS of the message service. The service reads TELNYX_API_KEY; the stored value is not used.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
