// Package configkeys declares the configuration keys of bin-tts-manager.
//
// The three keys are the provider credentials. The service reads them from the environment
// variables named below, and only from there; the stored value of a key is not read. The two AWS
// keys are independent secrets on purpose: the admin API writes one secret per call and each key is
// one row. The consumer refuses an AWS pair of which only one variable is set (see
// ttsconfig.AWSCredentials).
package configkeys

import (
	"monorepo/bin-common-handler/pkg/confighandler"
)

// Names of the tables of the configs.
const (
	Table        = "tts_configs"
	HistoryTable = "tts_config_histories"
)

// Names of the keys. A name never changes its kind: a stored row and its history would keep the
// old meaning. To make a key secret, declare a new key name.
const (
	// KeyAWSAccessKey is the AWS access key id of Amazon Polly.
	KeyAWSAccessKey = "aws.access_key"

	// KeyAWSSecretKey is the AWS secret access key of Amazon Polly. It is used together with
	// KeyAWSAccessKey.
	KeyAWSSecretKey = "aws.secret_key"

	// KeyElevenlabsAPIKey is the API key of the ElevenLabs streaming service.
	KeyElevenlabsAPIKey = "elevenlabs.api_key"
)

// Names of the environment variables that the service reads.
const (
	// EnvAWSAccessKey is the variable that holds the AWS access key id of Amazon Polly.
	EnvAWSAccessKey = "AWS_ACCESS_KEY"

	// EnvAWSSecretKey is the variable that holds the AWS secret access key of Amazon Polly.
	EnvAWSSecretKey = "AWS_SECRET_KEY"

	// EnvElevenlabsAPIKey is the variable that holds the ElevenLabs API key.
	EnvElevenlabsAPIKey = "ELEVENLABS_API_KEY"
)

// EnvName returns the variable that the service reads for the key, or an empty text for an unknown
// key.
func EnvName(key string) string {
	switch key {
	case KeyAWSAccessKey:
		return EnvAWSAccessKey
	case KeyAWSSecretKey:
		return EnvAWSSecretKey
	case KeyElevenlabsAPIKey:
		return EnvElevenlabsAPIKey
	default:
		return ""
	}
}

// maxSecretLen is the longest accepted key in bytes.
const maxSecretLen = 512

// Keys returns the declarations of the stored settings. A key has no environment name (Env stays
// empty): the handler of the stored settings never reads the environment, and the service reads the
// variables above directly.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyAWSAccessKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "AWS access key id used for Amazon Polly (batch fallback and streaming). The service reads AWS_ACCESS_KEY; the stored value is not used.", MaxLen: maxSecretLen},
		{Name: KeyAWSSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "AWS secret access key used for Amazon Polly (batch fallback and streaming). The service reads AWS_SECRET_KEY; the stored value is not used.", MaxLen: maxSecretLen},
		{Name: KeyElevenlabsAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the ElevenLabs streaming service. The service reads ELEVENLABS_API_KEY; the stored value is not used.", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
