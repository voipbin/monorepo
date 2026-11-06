package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Test_EnvValue(t *testing.T) {
	c := &Config{SendgridAPIKey: "sg-test", MailgunAPIKey: "mg-test", MailgunWebhookSigningKey: "", ConfigEncryptionKeys: "1:x"}

	tests := []struct {
		name string

		envName string

		expectValue string
		expectOK    bool
	}{
		{name: "sendgrid", envName: "SENDGRID_API_KEY", expectValue: "sg-test", expectOK: true},
		{name: "mailgun", envName: "MAILGUN_API_KEY", expectValue: "mg-test", expectOK: true},
		{name: "empty value is not set", envName: "MAILGUN_WEBHOOK_SIGNING_KEY", expectValue: "", expectOK: false},
		{name: "the master key list is not served to the registry", envName: "EMAIL_CONFIG_ENCRYPTION_KEYS", expectOK: false},
		{name: "unknown name", envName: "DATABASE_DSN", expectOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, ok := c.EnvValue(tt.envName)
			if v != tt.expectValue || ok != tt.expectOK {
				t.Errorf("Wrong match. expect: %q %v, got: %q %v", tt.expectValue, tt.expectOK, v, ok)
			}
		})
	}
}

// Test_masterKeyIsEnvOnly checks that the master key list is read from the prefixed variable only,
// that an unprefixed variable is ignored, and that no command line flag exists for it.
func Test_masterKeyIsEnvOnly(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}

	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")
	t.Setenv("EMAIL_CONFIG_ENCRYPTION_KEYS", "1:prefixed-value")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "1:prefixed-value" {
		t.Errorf("Wrong match. expect: the prefixed value, got: %q", got)
	}
	for _, name := range []string{"email_config_encryption_keys", "config_encryption_keys"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists for the master key list", name)
		}
	}
}

func Test_masterKeyUnprefixedIgnored(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}

	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "" {
		t.Errorf("Wrong match. expect: empty, got: %q", got)
	}
}
