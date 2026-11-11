// Package configkeys declares the configuration keys of bin-tts-manager.
//
// The three keys are the provider credentials that were read from the environment only. The two
// AWS keys are independent secrets on purpose: the admin API writes one secret per call and each
// key has one environment name. The consumer refuses a pair whose two keys come from different
// sources (see ttsconfig.AWSCredentials).
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

	// KeyAWSSecretKey is the AWS secret access key of Amazon Polly. It is stored together with
	// KeyAWSAccessKey.
	KeyAWSSecretKey = "aws.secret_key"

	// KeyElevenlabsAPIKey is the API key of the ElevenLabs streaming service.
	KeyElevenlabsAPIKey = "elevenlabs.api_key"
)

// maxSecretLen is the longest accepted key in bytes.
const maxSecretLen = 512

// Keys returns the declarations. The environment names are the variables that the service read
// before the configs existed, so a database without rows behaves exactly as before.
func Keys() []confighandler.Key {
	return []confighandler.Key{
		{Name: KeyAWSAccessKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "AWS access key id used for Amazon Polly (batch fallback and streaming). Store it together with aws.secret_key.", Env: "AWS_ACCESS_KEY", MaxLen: maxSecretLen},
		{Name: KeyAWSSecretKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "AWS secret access key used for Amazon Polly (batch fallback and streaming). Store it together with aws.access_key.", Env: "AWS_SECRET_KEY", MaxLen: maxSecretLen},
		{Name: KeyElevenlabsAPIKey, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "API key of the ElevenLabs streaming service.", Env: "ELEVENLABS_API_KEY", MaxLen: maxSecretLen},
	}
}

// NewRegistry builds the registry of the service.
func NewRegistry() (*confighandler.Registry, error) {
	return confighandler.NewRegistry(Keys())
}
