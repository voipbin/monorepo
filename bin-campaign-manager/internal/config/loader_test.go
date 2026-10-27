package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetLoaded makes the next Load read viper again and restores the loaded state afterwards.
func resetLoaded(t *testing.T) {
	t.Helper()

	saved := cfg
	cfg = nil
	once = sync.Once{}

	t.Cleanup(func() {
		viper.Reset()
		cfg = saved
		once = sync.Once{}
		if saved != nil {
			once.Do(func() {})
		}
	})
}

func Test_Get_loadsWhenNotLoaded(t *testing.T) {
	resetLoaded(t)
	viper.Set("database_dsn", "fake-dsn-a")
	viper.Set("redis_database", 7)

	res := Get()
	if res == nil || res.DatabaseDSN != "fake-dsn-a" || res.RedisDatabase != 7 {
		t.Fatalf("Wrong match. expect: the viper values, got: %+v", res)
	}

	// the loaded value is kept, a later change of viper is not read again
	viper.Set("database_dsn", "fake-dsn-b")
	if again := Get(); again != res || again.DatabaseDSN != "fake-dsn-a" {
		t.Errorf("Wrong match. expect: the same loaded config, got: %+v", again)
	}
}

func Test_LoadGlobalConfig_readsViper(t *testing.T) {
	resetLoaded(t)
	viper.Set("rabbitmq_address", "fake-rabbit-address")

	LoadGlobalConfig()

	if cfg == nil || cfg.RabbitMQAddress != "fake-rabbit-address" {
		t.Errorf("Wrong match. expect: rabbitmq address loaded, got: %+v", cfg)
	}
}

func Test_Bootstrap_registersFlags(t *testing.T) {
	resetLoaded(t)
	cmd := &cobra.Command{Use: "test"}

	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	f := cmd.Flags().Lookup("redis_database")
	if f == nil || f.DefValue != "1" {
		t.Errorf("Wrong match. expect: redis_database flag with default 1, got: %v", f)
	}
}
