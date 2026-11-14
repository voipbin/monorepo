// Package configkeys names the environment variables of the provider tokens of
// bin-message-manager.
//
// The two tokens are the authentication tokens of the SMS and MMS providers. The service reads them
// from the environment variables named below, and only from there. They are independent: each
// provider has its own token and they are rotated separately. The Telnyx variable (TELNYX_API_KEY)
// is the same one that bin-number-manager reads, so one rotation covers both services.
package configkeys

// Names of the environment variables that the service reads.
const (
	// EnvMessagebirdAPIKey is the variable that holds the MessageBird access key.
	EnvMessagebirdAPIKey = "MESSAGEBIRD_API_KEY"

	// EnvTelnyxAPIKey is the variable that holds the Telnyx API key. It is shared with
	// bin-number-manager.
	EnvTelnyxAPIKey = "TELNYX_API_KEY"
)
