package apierror

// Canonical wire strings for account-status refusals.
//
// These live here, next to EnvelopeFor, rather than in lib/middleware because
// two independent layers emit them and must emit them identically: the shared
// v1 authentication gate (lib/middleware) and POST /auth/login
// (lib/service/auth.go). A client that branches on the login refusal has to be
// able to reuse the same handling for the gate refusal, so the two responses
// are byte-identical by construction, not by convention.
//
// lib/apierror is the neutral home for that: it already owns the external
// error envelope's shape and is imported by both layers, so neither the
// handler layer nor the middleware layer has to depend on the other for the
// vocabulary they share.
const (
	// ReasonAccountExpired / MessageAccountExpired / RecoveryEndpointAccountExpired
	// describe a status='expired' refusal (VOIP-1491).
	//
	// The message deliberately does NOT promise that re-signing-up works.
	// For accounts expired after VOIP-1490 the customer row keeps tm_delete
	// NULL, so validateCreate's (deleted=false, email) duplicate check rejects
	// a re-signup; promising it would be a lie to that cohort (design §4-3).
	ReasonAccountExpired  = "ACCOUNT_EXPIRED"
	MessageAccountExpired = "This account has expired because its email address was never verified. " +
		"Request a new verification email, and contact support@voipbin.net if that does not resolve it."
	RecoveryEndpointAccountExpired = "POST /auth/email-verify-resend"

	// ReasonAccountDeleted / MessageAccountDeleted describe a status='deleted'
	// refusal. There is no recovery endpoint to advertise, so unlike expired
	// these carry no details payload.
	ReasonAccountDeleted  = "ACCOUNT_DELETED"
	MessageAccountDeleted = "This account has been deleted."
)

// AccountExpiredDetails returns the details payload carried by every
// ACCOUNT_EXPIRED refusal. A fresh slice is built per call so a caller that
// mutates the returned value cannot corrupt later responses.
func AccountExpiredDetails() []map[string]any {
	return []map[string]any{
		{
			"recovery_endpoint": RecoveryEndpointAccountExpired,
		},
	}
}
