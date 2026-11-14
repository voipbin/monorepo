// Package ttsconfig wires the configuration of bin-tts-manager: the suppliers that hand the
// provider credentials to the handlers at the time of use. The credentials come from the
// environment only.
package ttsconfig

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-tts-manager/internal/configkeys"
)

// ErrKeyUnavailable is returned when a provider key is refused: an AWS pair of which only one
// variable is set. It is never replaced by an empty value. A key that is simply not configured is
// not an error (an empty value). The error holds no key text.
var ErrKeyUnavailable = errors.New("the TTS provider key could not be read")

// ErrAWSPairIncomplete is wrapped by ErrKeyUnavailable when exactly one of the two AWS variables is
// set: a half pair is refused.
var ErrAWSPairIncomplete = errors.New("only one of the AWS access key and secret key is set")

// KeyFunc hands one provider key to a handler at the time of use. An empty value with no error
// means "not configured".
type KeyFunc func(ctx context.Context) (string, error)

// AWSCredentialsFunc hands the AWS key pair to a handler at the time of use. Empty values with no
// error mean "not configured".
type AWSCredentialsFunc func(ctx context.Context) (accessKey, secretKey string, err error)

// ElevenlabsKey returns the supplier of the ElevenLabs key. The key is the value of
// ELEVENLABS_API_KEY at the time of use; the supplier keeps nothing. A variable that is not set, is empty or holds an unresolved placeholder is not configured:
// an empty value and no error.
func ElevenlabsKey() KeyFunc {
	return func(ctx context.Context) (string, error) {
		value, _ := envsecret.Lookup(configkeys.EnvElevenlabsAPIKey)

		return value, nil
	}
}

// AWSCredentials returns the supplier of the AWS key pair. It reads both variables on every call
// and keeps nothing. Both set: the pair. Neither set (not set,
// empty or an unresolved placeholder): not configured, empty values and no error. Exactly one
// set: ErrKeyUnavailable wrapping ErrAWSPairIncomplete, a half pair is refused.
func AWSCredentials() AWSCredentialsFunc {
	return func(ctx context.Context) (string, string, error) {
		access, _ := envsecret.Lookup(configkeys.EnvAWSAccessKey)
		secret, _ := envsecret.Lookup(configkeys.EnvAWSSecretKey)

		if (access == "") != (secret == "") {
			return "", "", fmt.Errorf("%w: %w", ErrKeyUnavailable, ErrAWSPairIncomplete)
		}

		return access, secret, nil
	}
}

// LogStates logs, once, the state of each variable (name and state only, never a value). A variable
// that is not set is an info line, because every provider key is optional; a placeholder (the text
// of an unresolved deployment variable) is a warning. When exactly one of the two AWS variables is
// set, one more warning tells that the AWS credentials are refused.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	awsSet := 0
	for _, name := range []string{configkeys.EnvAWSAccessKey, configkeys.EnvAWSSecretKey, configkeys.EnvElevenlabsAPIKey} {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch state {
		case envsecret.Set:
			if name != configkeys.EnvElevenlabsAPIKey {
				awsSet++
			}
			entry.Infof("The variable is set. variable: %s", name)
		case envsecret.Placeholder:
			entry.Warnf("The variable holds an unresolved placeholder. variable: %s", name)
		default:
			entry.Infof("The variable is not set. variable: %s", name)
		}
	}

	if awsSet == 1 {
		log.Warnf("Only one of the two AWS variables is set, so the AWS credentials are refused. Set or unset both. variables: %s, %s", configkeys.EnvAWSAccessKey, configkeys.EnvAWSSecretKey)
	}
}
