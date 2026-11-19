// Package messageconfig wires the configuration of bin-message-manager: the suppliers that hand a
// provider token to the request layer at the time of use. The tokens come from the environment only.
package messageconfig

import (
	"context"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-message-manager/internal/configkeys"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// TokenFunc hands a provider token to the request layer at the time of use. An empty token with
// no error means that the token is not configured.
type TokenFunc = func(ctx context.Context) (string, error)

// MessagebirdToken returns the supplier of the MessageBird token. The token is the value of
// MESSAGEBIRD_API_KEY at the time of use.
func MessagebirdToken() TokenFunc {
	return token(configkeys.EnvMessagebirdAPIKey)
}

// TelnyxToken returns the supplier of the Telnyx token (TELNYX_API_KEY, the variable that
// bin-number-manager reads too).
func TelnyxToken() TokenFunc {
	return token(configkeys.EnvTelnyxAPIKey)
}

// TwilioCredentials returns the supplier of the Twilio Account SID and Auth Token (TWILIO_ACCOUNT_SID and
// TWILIO_AUTH_TOKEN, the variables that bin-number-manager reads too). Both values are empty unless both
// variables are configured: a Twilio request needs the pair, so half a pair is the same as none.
func TwilioCredentials() func(ctx context.Context) (sid string, token string, err error) {
	return func(ctx context.Context) (string, string, error) {
		sid, _ := envsecret.Lookup(configkeys.EnvTwilioAccountSID)
		token, _ := envsecret.Lookup(configkeys.EnvTwilioAuthToken)
		if sid == "" || token == "" {
			return "", "", nil
		}

		return sid, token, nil
	}
}

// Twilio returns the Twilio settings of the request layer: the credentials and the status callback URL.
func Twilio() requestexternal.TwilioConfig {
	return requestexternal.TwilioConfig{
		Credentials: TwilioCredentials(),
		CallbackURL: TwilioStatusCallbackURL(),
	}
}

// twilioCallbackPathSuffix is the end of the hook route that receives the Twilio callbacks.
const twilioCallbackPathSuffix = "/messages/twilio"

// TwilioStatusCallbackURL returns the supplier of the Twilio status callback URL
// (TWILIO_STATUS_CALLBACK_URL). The value is empty when the variable is not set, is a placeholder or is not a
// valid callback URL (see ValidCallbackURL). The request that sends a message and the check of the signature
// of a callback use this same supplier, so a value that is not valid counts as not set in both.
func TwilioStatusCallbackURL() func(ctx context.Context) string {
	return func(ctx context.Context) string {
		value, _ := envsecret.Lookup(configkeys.EnvTwilioStatusCallbackURL)
		if !ValidCallbackURL(value) {
			return ""
		}

		return value
	}
}

// ValidCallbackURL tells whether the text can be the Twilio status callback URL: an http or https URL with a
// host, no user information, no query string, no fragment, no explicit default port (the string that Twilio
// signs could then differ from the text), and a path that ends with /messages/twilio, the route of the
// hook-manager (any other path would be dropped as an unknown hook).
func ValidCallbackURL(value string) bool {
	if value == "" {
		return false
	}

	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(value, "#") {
		return false
	}
	if port := u.Port(); (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		return false
	}

	return strings.HasSuffix(u.Path, twilioCallbackPathSuffix)
}

// token returns the function that reads one variable on every call and keeps nothing, so a changed
// variable is used by the next call after a restart of the replica. A variable that is not set,
// is empty or holds an unresolved placeholder is not configured: an empty token and no error.
func token(name string) TokenFunc {
	return func(ctx context.Context) (string, error) {
		value, _ := envsecret.Lookup(name)

		return value, nil
	}
}

// LogStates logs, once, the state of each variable (name and state only, never a value). A Telnyx
// variable that is not set is a warning, because Telnyx is the first provider of a send from a Telnyx
// number or from a number without an owner; a MessageBird or Twilio variable that is not set is an info
// line, because those providers serve their own numbers and the targets that were refused before. A
// placeholder (the text of an unresolved deployment variable) is a warning. A callback URL that is set but
// not valid is a warning too, because the callbacks of Twilio would be refused.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range []string{
		configkeys.EnvMessagebirdAPIKey,
		configkeys.EnvTelnyxAPIKey,
		configkeys.EnvTwilioAccountSID,
		configkeys.EnvTwilioAuthToken,
		configkeys.EnvTwilioStatusCallbackURL,
	} {
		value, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch state {
		case envsecret.Set:
			if name == configkeys.EnvTwilioStatusCallbackURL && !ValidCallbackURL(value) {
				entry.Warnf("The variable is set but it is not a valid callback URL, so it is ignored. variable: %s", name)
				continue
			}
			entry.Infof("The variable is set. variable: %s", name)
		case envsecret.Placeholder:
			entry.Warnf("The variable holds an unresolved placeholder. variable: %s", name)
		default:
			if name == configkeys.EnvTelnyxAPIKey {
				entry.Warnf("The variable is not set. variable: %s", name)
			} else {
				entry.Infof("The variable is not set. variable: %s", name)
			}
		}
	}
}
