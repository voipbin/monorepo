// Package configkeys names the environment variables of the provider credentials of
// bin-email-manager.
//
// The three credentials are independent on purpose: the Mailgun API key (sending) and the Mailgun
// webhook signing key (receiving) have different purposes and are rotated separately. The service
// reads them from environment variables (the Env constants below).
package configkeys

// Names of the environment variables that the service reads. A variable that is unset, empty or an
// unresolved placeholder means that the credential is not configured.
const (
	EnvSendgridAPIKey           = "SENDGRID_API_KEY"
	EnvMailgunAPIKey            = "MAILGUN_API_KEY"
	EnvMailgunWebhookSigningKey = "MAILGUN_WEBHOOK_SIGNING_KEY"
)
