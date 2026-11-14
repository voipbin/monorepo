// Package transcribeconfig wires the configuration of bin-transcribe-manager: the supplier that
// hands the AWS credential to the streaming handler at the time of use. The credential comes from
// the environment only.
package transcribeconfig

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/envsecret"

	"monorepo/bin-transcribe-manager/internal/configkeys"
)

// AWSKeysFunc hands the AWS credential to the streaming handler at the time of use. An empty
// pair with no error means "AWS is not available" (not configured, or only one of the two variables
// is set).
type AWSKeysFunc = func(ctx context.Context) (accessKey, secretKey string, err error)

// AWSKeys returns the function that hands the AWS credential to the streaming handler at the
// time of use. It reads both variables on every call and keeps nothing.
//
// The two variables are one credential, so a pair is handed out only when both hold a value. Every
// other state is "AWS is not available" (an empty pair, no error): neither variable (not set, empty
// or an unresolved placeholder), or one variable only. The half pair is logged once at start by
// LogStates; nothing is logged per call.
func AWSKeys() AWSKeysFunc {
	return func(ctx context.Context) (string, string, error) {
		accessKey, _ := envsecret.Lookup(configkeys.EnvAWSAccessKey)
		secretKey, _ := envsecret.Lookup(configkeys.EnvAWSSecretKey)

		if accessKey == "" || secretKey == "" {
			return "", "", nil
		}

		return accessKey, secretKey, nil
	}
}

// LogStates logs, once, the state of each variable (name and state only, never a value). A variable
// that is not set is an info line, because AWS is an optional provider; a placeholder (the text of
// an unresolved deployment variable) is a warning. When exactly one of the two variables is set, one
// more warning tells that AWS is not available.
func LogStates() {
	log := logrus.WithField("func", "LogStates")

	set := 0
	for _, name := range []string{configkeys.EnvAWSAccessKey, configkeys.EnvAWSSecretKey} {
		_, state := envsecret.Lookup(name)
		entry := log.WithFields(logrus.Fields{"variable": name, "state": state.String()})

		switch state {
		case envsecret.Set:
			set++
			entry.Infof("The variable is set. variable: %s", name)
		case envsecret.Placeholder:
			entry.Warnf("The variable holds an unresolved placeholder. variable: %s", name)
		default:
			entry.Infof("The variable is not set. variable: %s", name)
		}
	}

	if set == 1 {
		log.Warnf("Only one of the two AWS variables is set, so AWS is not available. variables: %s, %s", configkeys.EnvAWSAccessKey, configkeys.EnvAWSSecretKey)
	}
}
