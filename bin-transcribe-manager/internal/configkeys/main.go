// Package configkeys declares the configuration keys of bin-transcribe-manager.
//
// The two keys are the AWS credential of the AWS Transcribe provider. The service reads the
// credential from the environment variables named below, and only from there; the stored value of a
// key is not read. They are independent secret keys: the shared library can not declare an atomic
// group with two secrets (its anchor must be one secret), and the admin API and the command line
// write secrets only. That the two values belong together is enforced where they are read
// (transcribeconfig.AWSKeys): a pair is used only when both variables hold a value.
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "transcribe_configs"
	HistoryTable = "transcribe_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To make a key secret, declare a new key name.
const (
	// KeyAWSAccessKey is the access key id of the AWS account that serves AWS Transcribe. It is an
	// identifier in the terms of AWS, and is declared secret so that no value reaches the admin
	// API, the history or a log.
	KeyAWSAccessKey = "aws.access_key"

	// KeyAWSSecretKey is the secret access key that belongs to KeyAWSAccessKey.
	KeyAWSSecretKey = "aws.secret_key"
)

// Names of the environment variables that the service reads.
const (
	// EnvAWSAccessKey is the variable that holds the AWS access key id.
	EnvAWSAccessKey = "AWS_ACCESS_KEY"

	// EnvAWSSecretKey is the variable that holds the AWS secret access key.
	EnvAWSSecretKey = "AWS_SECRET_KEY"
)

// EnvName returns the variable that the service reads for the key, or an empty text for an unknown
// key.
func EnvName(key string) string {
	switch key {
	case KeyAWSAccessKey:
		return EnvAWSAccessKey
	case KeyAWSSecretKey:
		return EnvAWSSecretKey
	default:
		return ""
	}
}

// maxSecretLen is the longest accepted AWS credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations of the stored settings. A key has no environment name (Env stays
// empty): the handler of the stored settings never reads the environment, and the service reads the
// variables above directly.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyAWSAccessKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Access key id of the AWS account that serves AWS Transcribe. The service reads AWS_ACCESS_KEY, together with the secret access key; the stored value is not used.", MaxLen: maxSecretLen},
		{Name: KeyAWSSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Secret access key of the AWS account that serves AWS Transcribe. The service reads AWS_SECRET_KEY, together with the access key id; the stored value is not used.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
