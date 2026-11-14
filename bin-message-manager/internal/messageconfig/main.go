// Package messageconfig wires the configuration of bin-message-manager: the suppliers that hand a
// provider token to the request layer at the time of use. The tokens come from the environment only.
package messageconfig

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-message-manager/internal/configkeys"
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
// variable that is not set is a warning, because Telnyx is the first provider of every send; a
// MessageBird variable that is not set is an info line, because MessageBird only serves the targets
// that Telnyx refused. A placeholder (the text of an unresolved deployment variable) is a warning.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range []string{configkeys.EnvMessagebirdAPIKey, configkeys.EnvTelnyxAPIKey} {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch state {
		case envsecret.Set:
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
