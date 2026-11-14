// Package configkeys names the environment variables of the provider credentials of
// bin-tts-manager.
//
// The three credentials are read from the environment variables named below, and only from there.
// The two AWS variables are one pair: the consumer refuses an AWS pair of which only one variable
// is set (see ttsconfig.AWSCredentials).
package configkeys

// Names of the environment variables that the service reads.
const (
	// EnvAWSAccessKey is the variable that holds the AWS access key id of Amazon Polly.
	EnvAWSAccessKey = "AWS_ACCESS_KEY"

	// EnvAWSSecretKey is the variable that holds the AWS secret access key of Amazon Polly.
	EnvAWSSecretKey = "AWS_SECRET_KEY"

	// EnvElevenlabsAPIKey is the variable that holds the ElevenLabs API key.
	EnvElevenlabsAPIKey = "ELEVENLABS_API_KEY"
)
