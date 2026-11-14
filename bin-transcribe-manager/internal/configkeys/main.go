// Package configkeys names the environment variables of the AWS credential of
// bin-transcribe-manager.
//
// The two variables are the AWS credential of the AWS Transcribe provider. The service reads the
// credential from the environment variables named below, and only from there. That the two values
// belong together is enforced where they are read (transcribeconfig.AWSKeys): a pair is used only
// when both variables hold a value.
package configkeys

// Names of the environment variables that the service reads.
const (
	// EnvAWSAccessKey is the variable that holds the AWS access key id.
	EnvAWSAccessKey = "AWS_ACCESS_KEY"

	// EnvAWSSecretKey is the variable that holds the AWS secret access key.
	EnvAWSSecretKey = "AWS_SECRET_KEY"
)
