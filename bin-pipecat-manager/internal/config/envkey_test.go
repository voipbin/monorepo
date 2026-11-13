package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func loadMasterKeyForTest(t *testing.T) (*cobra.Command, Config) {
	t.Helper()

	restoreLogger(t)
	viper.Reset()
	prevConfig := appConfig
	t.Cleanup(func() {
		viper.Reset()
		appConfig = prevConfig
		once = sync.Once{}
	})
	once = sync.Once{}

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	return cmd, Get()
}

// The master key list is read from the prefixed variable only, an unprefixed variable is ignored,
// and no command line flag exists for it (a flag value shows in the process list).
func Test_masterKeyIsEnvOnly(t *testing.T) {
	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")
	t.Setenv("PIPECAT_CONFIG_ENCRYPTION_KEYS", "1:prefixed-value")

	cmd, got := loadMasterKeyForTest(t)

	if got.ConfigEncryptionKeys != "1:prefixed-value" {
		t.Errorf("Wrong match. expect: the prefixed value, got: %q", got.ConfigEncryptionKeys)
	}
	for _, name := range []string{"pipecat_config_encryption_keys", "config_encryption_keys"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists for the master key list", name)
		}
	}
}

func Test_masterKeyUnprefixedIgnored(t *testing.T) {
	t.Setenv("CONFIG_ENCRYPTION_KEYS", "9:unprefixed-must-be-ignored")

	_, got := loadMasterKeyForTest(t)

	if got.ConfigEncryptionKeys != "" {
		t.Errorf("Wrong match. expect: empty, got: %q", got.ConfigEncryptionKeys)
	}
}
