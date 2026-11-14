// Package transcribeconfig wires the configuration of bin-transcribe-manager: the handler of the
// stored settings (the master key and the registry, used by the config commands and the admin API)
// and the supplier that hands the AWS credential to the streaming handler at the time of use. The
// credential comes from the environment only; the stored settings are not read.
package transcribeconfig

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/envsecret"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/internal/configkeys"
)

// maxSecretPlaintext is the longest secret the cipher accepts. It is the second guard behind the
// MaxLen of each key and fits the column (2048 bytes minus nonce and tag).
const maxSecretPlaintext = 2020

// ErrKeyUnavailable stays for the stored settings code until its removal. The environment read can
// not fail, so AWSKeys never returns it.
var ErrKeyUnavailable = errors.New("the AWS credential could not be read")

// AWSKeysFunc hands the AWS credential to the streaming handler at the time of use. An empty
// pair with no error means "AWS is not available" (not configured, or only one of the two variables
// is set).
type AWSKeysFunc = func(ctx context.Context) (accessKey, secretKey string, err error)

// New creates the config handler of the service on the given store.
//
// An unparsable master key list never stops the service: the error is logged (without any key
// text) and the handler works without a master key. Secrets can not be written then, and a stored
// secret is reported as undecryptable.
// The only error is an invalid registry, which is a programming error.
func New(cfg *config.Config, store confighandler.Store, opts ...confighandler.Option) (confighandler.Handler, error) {
	reg, err := configkeys.NewRegistry()
	if err != nil {
		return nil, errors.Wrap(err, "could not create the config registry")
	}

	crypto, errCrypto := secretcrypto.New(cfg.ConfigEncryptionKeys, secretcrypto.WithMaxPlaintext(maxSecretPlaintext))
	if errCrypto != nil {
		logrus.WithField("func", "New").Errorf("The %s value is invalid, so stored secrets can not be used. err: %v", config.EnvConfigEncryptionKeys, errCrypto)
		crypto, _ = secretcrypto.New("", secretcrypto.WithMaxPlaintext(maxSecretPlaintext))
	}

	return confighandler.NewHandler(reg, store, crypto, nil, opts...), nil
}

// AWSKeys returns the function that hands the AWS credential to the streaming handler at the
// time of use. It reads both variables on every call and keeps nothing; the stored settings are
// not read.
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
