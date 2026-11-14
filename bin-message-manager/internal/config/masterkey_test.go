package config

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetLoadState gives the test a fresh once and config, and restores a fresh once and the
// previous config afterwards, so the singleton does not leak across tests.
func resetLoadState(t *testing.T) {
	t.Helper()

	saved := globalConfig
	once = sync.Once{}
	globalConfig = Config{}
	viper.Reset()

	t.Cleanup(func() {
		once = sync.Once{}
		globalConfig = saved
		viper.Reset()
	})
}

// Test_Config_removedFields checks that the provider token fields and the EnvValue method are gone:
// the tokens are read from the environment at the time of use, never held by the process configuration.
func Test_Config_removedFields(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	for _, name := range []string{"AuthtokenMessagebird", "AuthtokenTelnyx"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("Wrong match. the field %s must not exist", name)
		}
	}
	if _, ok := reflect.TypeOf(&Config{}).MethodByName("EnvValue"); ok {
		t.Errorf("Wrong match. the method EnvValue must not exist")
	}
}

// Test_providerVariablesAreNotLoaded checks that the provider token variables are read at the time
// of use by messageconfig and are not part of the process configuration: the load succeeds with
// them exported, no flag exists for them, no value reaches the Config, and the
// other fields are not affected.
func Test_providerVariablesAreNotLoaded(t *testing.T) {
	resetLoadState(t)

	t.Setenv("MESSAGEBIRD_API_KEY", "provider-fake-mb")
	t.Setenv("TELNYX_API_KEY", "provider-fake-tn")
	t.Setenv("DATABASE_DSN", "fake-dsn")
	t.Setenv("REDIS_DATABASE", "3")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	cfg := Get()
	if cfg.DatabaseDSN != "fake-dsn" || cfg.RedisDatabase != 3 {
		t.Errorf("Wrong match. the other fields are affected: %+v", cfg)
	}
	for _, name := range []string{"messagebird_api_key", "telnyx_api_key", "authtoken_messagebird", "authtoken_telnyx"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists", name)
		}
	}
	// none of the provider values is held by the config
	if s := fmt.Sprintf("%+v", *cfg); strings.Contains(s, "provider-fake") {
		t.Errorf("Wrong match. a provider value reached the config: %s", s)
	}
}
