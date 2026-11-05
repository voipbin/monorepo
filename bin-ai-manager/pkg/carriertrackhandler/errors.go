package carriertrackhandler

import (
	"strings"
	"time"

	"monorepo/bin-ai-manager/models/mcpserver"
)

// ErrCode is the outcome of a lookup that did not return a result. Every code
// has one fixed text for the model; no text repeats the input, a key or what
// a carrier answered.
type ErrCode string

// The error codes.
const (
	CodeCredentialNotConfigured ErrCode = "credential_not_configured"
	CodeCredentialMalformed     ErrCode = "credential_malformed"
	CodeCredentialInvalid       ErrCode = "credential_invalid"
	CodeRateLimited             ErrCode = "rate_limited"
	CodeCarrierUnavailable      ErrCode = "carrier_unavailable"
	CodeServiceBusy             ErrCode = "service_busy"
	CodeInvalidFormat           ErrCode = "invalid_format"
	CodeNotFound                ErrCode = "not_found"
	CodeToolUnknown             ErrCode = "tool_unknown"
)

const (
	// cooldownCredentialInvalid is how long a rejected key is not tried again.
	cooldownCredentialInvalid = 10 * time.Minute
	// cooldownRateLimited is how long a rate limited key is not tried again.
	cooldownRateLimited = time.Minute
)

// CarrierError is what an adapter reports. It carries no text of the carrier.
// It is deliberately not an error: a nil *CarrierError must never turn into a
// non-nil error interface.
type CarrierError struct {
	Code ErrCode

	// Cooldown overrides the default cooldown of the code. It is applied to
	// the (customer, carrier, key) that made the call.
	Cooldown time.Duration

	// DropToken discards the cached access token of this call.
	DropToken bool
}

// cooldown returns how long the key is blocked after this error.
func (e *CarrierError) cooldown() time.Duration {
	if e.Cooldown > 0 {
		return e.Cooldown
	}
	switch e.Code {
	case CodeCredentialInvalid:
		return cooldownCredentialInvalid
	case CodeRateLimited:
		return cooldownRateLimited
	}
	return 0
}

// displayName is how a carrier is named in a text.
func displayName(carrier string) string {
	switch carrier {
	case "ups":
		return "UPS"
	case "fedex":
		return "FedEx"
	case "dhl":
		return "DHL"
	}
	return strings.ToUpper(carrier)
}

// Message returns the fixed text for code. The model sees it as the result of
// the tool call, at most 200 bytes of it.
func Message(code ErrCode, carrier string, format mcpserver.CredentialFormat) string {
	c := displayName(carrier)

	switch code {
	case CodeCredentialNotConfigured:
		return "No " + c + " API key is saved in this MCP server's settings. Tell the agent that an account admin must add it. Do not retry."

	case CodeCredentialMalformed:
		if format == mcpserver.CredentialClientIDSecret {
			return "The saved " + c + " API key has the wrong format. Tell the agent that an account admin must re-enter it as CLIENT_ID:CLIENT_SECRET. Do not retry."
		}
		return "The saved " + c + " API key has an invalid format. Tell the agent that an account admin must re-enter the " + c + " API key. Do not retry."

	case CodeCredentialInvalid:
		return c + " rejected the saved API key. Tell the agent that an account admin must re-enter a production key in this MCP server's settings. Do not retry."

	case CodeRateLimited:
		return "The " + c + " lookup limit for this account is reached. Tell the agent to check the carrier website or try again later. Do not retry during this call."

	case CodeCarrierUnavailable:
		return c + " tracking is temporarily unavailable. Tell the agent to try again later. Do not retry unless the caller asks again."

	case CodeServiceBusy:
		return "The tracking service is busy for " + c + " right now. Tell the agent to try again in a few minutes. Do not retry unless the caller asks again."

	case CodeInvalidFormat:
		if carrier == "ups" {
			return "This does not look like a UPS 1Z tracking number. Ask the caller to repeat it or confirm the carrier."
		}
		return "The tracking number does not look valid for " + c + ". Ask the caller to repeat it or confirm the carrier."

	case CodeNotFound:
		return c + " has no record of this tracking number. It may be mistyped or too new. Ask the caller to confirm it."

	case CodeToolUnknown:
		return "This MCP server does not offer that tool. Do not retry."
	}

	return c + " tracking is temporarily unavailable. Tell the agent to try again later. Do not retry unless the caller asks again."
}
