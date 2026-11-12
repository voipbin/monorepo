// Package configkeys declares the configuration keys of bin-transcribe-manager.
//
// The two keys are the AWS credential of the AWS Transcribe provider. They are stored in the
// database only. They are independent secret keys: the shared library can not declare an
// atomic group with two secrets (its anchor must be one secret), and the admin API and the
// command line write secrets only. That the two values belong together is enforced where they
// are read (transcribeconfig.AWSKeys): a pair is used only when both come from the same source.
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

// maxSecretLen is the longest accepted AWS credential in bytes.
const maxSecretLen = 512

// Keys returns the declarations. The values are stored in the database only: a key has no
// environment name, so a key without a row is not configured.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyAWSAccessKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Access key id of the AWS account that serves AWS Transcribe. It is used only together with the secret access key, from the same source.", MaxLen: maxSecretLen},
		{Name: KeyAWSSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "Secret access key of the AWS account that serves AWS Transcribe. It is used only together with the access key id, from the same source.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
