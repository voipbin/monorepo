// Package aiconfig wires the configuration of bin-ai-manager: the platform LLM keys and the OAuth
// app of each MCP vendor, which are read from environment variables when a call needs them.
package aiconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-ai-manager/internal/configkeys"
)

const (
	// geminiURLMarker marks the base URL of the Gemini OpenAI compatible endpoint. It is the rule
	// that picks the key of a provider, and it is written here only.
	geminiURLMarker = "generativelanguage"
)

// ErrKeyUnavailable is returned when a platform key could not be read. An environment read can not
// fail, so the suppliers of this package never return it; no producer remains, and builderhandler
// still maps it. The error holds no key text.
var ErrKeyUnavailable = errors.New("the platform LLM key could not be read")

// ErrMCPOAuthIncomplete is returned when the client_secret of an MCP OAuth app is set and the
// client_id is not (or is empty, or an unresolved placeholder), so the pair is not usable. The error
// holds no value.
var ErrMCPOAuthIncomplete = errors.New("the MCP OAuth app is configured incompletely")

// KeyFunc hands a platform key to a handler at the time of use. An empty value with no error means
// "not configured" (the variable is not set, is empty, or still holds an unresolved placeholder).
type KeyFunc = func(ctx context.Context) (string, error)

// Supplier returns the function that hands the value of the environment variable to a handler at
// the time of use. It reads the variable on every call and keeps nothing. A variable that is not
// set, is empty or still holds an unresolved placeholder gives an empty value and no error.
func Supplier(envName string) KeyFunc {
	return func(_ context.Context) (string, error) {
		value, _ := envsecret.Lookup(envName)

		return value, nil
	}
}

// MCPOAuthClient resolves the (client_id, client_secret) pair of the MCP OAuth app of a vendor from
// its environment variables.
//
// The client_secret is read first. With a client_secret, the client_id is required: without one
// the result is ErrMCPOAuthIncomplete. Without a client_secret the result is the empty pair ("not
// configured"), and so is a client_id alone. An unknown vendor is an error. No value is part of an
// error text.
func MCPOAuthClient(vendor string) (clientID, clientSecret string, err error) {
	idName, secretName, ok := configkeys.MCPOAuthEnvNames(vendor)
	if !ok {
		return "", "", fmt.Errorf("unknown MCP OAuth vendor: %q", vendor)
	}

	secret, _ := envsecret.Lookup(secretName)
	if secret == "" {
		return "", "", nil
	}

	id, _ := envsecret.Lookup(idName)
	if id == "" {
		return "", "", fmt.Errorf("%w: %s is not set", ErrMCPOAuthIncomplete, idName)
	}

	return id, secret, nil
}

// IsGeminiBaseURL tells whether a base URL is the Gemini OpenAI compatible endpoint.
func IsGeminiBaseURL(baseURL string) bool {
	return strings.Contains(baseURL, geminiURLMarker)
}

// KeyForBaseURL returns the supplier of the key that belongs to the provider at the base URL: the
// Gemini key for the Gemini endpoint, the OpenAI compatible key otherwise (the OpenAI default and
// any other compatible provider). It is the one place of this rule.
func KeyForBaseURL(baseURL string) KeyFunc {
	if IsGeminiBaseURL(baseURL) {
		return GeminiKey()
	}

	return Supplier(configkeys.EnvOpenAIAPIKey)
}

// GeminiKey returns the supplier of the Gemini key, for the handlers that always call the Gemini
// endpoint (audit and prompt proposal).
func GeminiKey() KeyFunc {
	return Supplier(configkeys.EnvGoogleAPIKey)
}

// Configured tells whether the key has a value now. A read error is returned as an error.
func Configured(ctx context.Context, f KeyFunc) (bool, error) {
	v, err := f(ctx)
	if err != nil {
		return false, err
	}

	return v != "", nil
}

// LogStates logs, once, the state of each environment variable of the platform keys (the variable
// name and the state only, never a value or a part of one). A variable that is not set is a warning,
// like a key that was not stored before, and so is an unresolved placeholder of the deployment. The
// service starts anyway.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	for _, name := range configkeys.EnvNames() {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"env": name, "state": state.String()})

		switch state {
		case envsecret.Set:
			entry.Infof("Platform key variable. env: %s, state: set", name)
		case envsecret.Placeholder:
			entry.Warnf("The platform key variable holds an unresolved placeholder, so it is not configured. env: %s", name)
		default:
			entry.Warnf("The platform key variable is not set, so the key is not configured. env: %s", name)
		}
	}
}
