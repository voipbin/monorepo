// Package configkeys names the environment variables of the provider tokens of
// bin-message-manager.
//
// The tokens are the authentication tokens of the SMS and MMS providers. The service reads them
// from the environment variables named below, and only from there. They are independent: each
// provider has its own token and they are rotated separately. The Telnyx variable (TELNYX_API_KEY)
// and the Twilio variables are the same ones that bin-number-manager reads, so one rotation covers
// both services.
package configkeys

// Names of the environment variables that the service reads.
const (
	// EnvMessagebirdAPIKey is the variable that holds the MessageBird access key.
	EnvMessagebirdAPIKey = "MESSAGEBIRD_API_KEY"

	// EnvTelnyxAPIKey is the variable that holds the Telnyx API key. It is shared with
	// bin-number-manager.
	EnvTelnyxAPIKey = "TELNYX_API_KEY"

	// EnvTwilioAccountSID and EnvTwilioAuthToken are the Twilio credentials. They are the same variables
	// that bin-number-manager reads. Twilio sending is available only when both are set.
	EnvTwilioAccountSID = "TWILIO_ACCOUNT_SID"
	EnvTwilioAuthToken  = "TWILIO_AUTH_TOKEN"

	// EnvTwilioStatusCallbackURL is the public URL of the Twilio delivery callback (the hook route
	// .../v1.0/messages/twilio of bin-hook-manager). It is given to Twilio with every request and it is the
	// URL that the signature of a callback is checked against. When it is not set the messages are sent
	// without a callback.
	EnvTwilioStatusCallbackURL = "TWILIO_STATUS_CALLBACK_URL"
)
