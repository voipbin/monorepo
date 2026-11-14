// Package configkeys names the environment variables of the Paddle credentials of
// bin-billing-manager.
//
// The two credentials are the API key that opens a customer portal session, and the secret that
// signs the webhooks of Paddle. They are independent secrets. The service reads each one from its
// environment variable (the Env... constants) at the time of use (see billingconfig).
package configkeys

// Names of the environment variables that the service reads. A variable that is unset, empty or an
// unresolved placeholder means that the credential is not configured.
const (
	// EnvPaddleAPIKey is the variable of the Paddle API key.
	EnvPaddleAPIKey = "PADDLE_API_KEY"

	// EnvPaddleWebhookSecret is the variable of the secret that signs the Paddle webhooks.
	EnvPaddleWebhookSecret = "PADDLE_WEBHOOK_SECRET"
)
