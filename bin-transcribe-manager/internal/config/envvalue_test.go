package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func Test_EnvValue(t *testing.T) {
	c := &Config{AWSAccessKey: "access-test", ConfigEncryptionKeys: "1:x"}

	tests := []struct {
		name string

		envName string

		expectValue string
		expectOK    bool
	}{
		{name: "access key", envName: "AWS_ACCESS_KEY", expectValue: "access-test", expectOK: true},
		{name: "empty value is not set", envName: "AWS_SECRET_KEY", expectValue: "", expectOK: false},
		{name: "the master key list is not served to the registry", envName: "TRANSCRIBE_CONFIG_ENCRYPTION_KEYS", expectOK: false},
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

// Test_EnvValue_flag checks that the registry sees the value that the direct read saw: a command
// line flag counts as well as the environment variable.
func Test_EnvValue_flag(t *testing.T) {
	resetLoadState(t)

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := cmd.PersistentFlags().Set("aws_access_key", "from-flag"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if v, ok := Get().EnvValue("AWS_ACCESS_KEY"); !ok || v != "from-flag" {
		t.Errorf("Wrong match. expect: the flag value, got: %q %v", v, ok)
	}
}

// Test_masterKeyIsEnvOnly checks that the master key list is read from the prefixed variable only,
// that an unprefixed variable is ignored, and that no command line flag exists for it.
func Test_masterKeyIsEnvOnly(t *testing.T) {
	resetLoadState(t)

	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")
	t.Setenv("TRANSCRIBE_CONFIG_ENCRYPTION_KEYS", "1:prefixed-value")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "1:prefixed-value" {
		t.Errorf("Wrong match. expect: the prefixed value, got: %q", got)
	}
	for _, name := range []string{"transcribe_config_encryption_keys", "config_encryption_keys"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists for the master key list", name)
		}
	}
}

func Test_masterKeyUnprefixedIgnored(t *testing.T) {
	resetLoadState(t)

	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().ConfigEncryptionKeys; got != "" {
		t.Errorf("Wrong match. expect: empty, got: %q", got)
	}
}
