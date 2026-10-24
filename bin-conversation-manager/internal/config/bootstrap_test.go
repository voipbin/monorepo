package config

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Test_Bootstrap(t *testing.T) {
	// Bootstrap changes the process wide viper and logrus state, so restore both afterwards
	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()
	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
	})

	// the environment overrides the defaults once the flags are bound
	t.Setenv("DATABASE_DSN", "user:pass@tcp(db.invalid:3306)/test")
	t.Setenv("REDIS_DATABASE", "5")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if got := logrus.GetLevel(); got != logrus.DebugLevel {
		t.Errorf("log level = %v, want %v", got, logrus.DebugLevel)
	}

	defaults := map[string]string{
		"rabbitmq_address":          "",
		"prometheus_endpoint":       "/metrics",
		"prometheus_listen_address": ":2112",
		"database_dsn":              "",
		"redis_address":             "",
		"redis_password":            "",
		"redis_database":            "0",
	}
	for name, want := range defaults {
		f := cmd.PersistentFlags().Lookup(name)
		if f == nil {
			t.Errorf("persistent flag %q is not registered", name)
			continue
		}
		if f.DefValue != want {
			t.Errorf("flag %q default = %q, want %q", name, f.DefValue, want)
		}
	}

	if got := viper.GetString("database_dsn"); got != "user:pass@tcp(db.invalid:3306)/test" {
		t.Errorf("database_dsn from the environment = %q", got)
	}
	if got := viper.GetInt("redis_database"); got != 5 {
		t.Errorf("redis_database from the environment = %d, want 5", got)
	}
	if got := viper.GetString("prometheus_listen_address"); got != ":2112" {
		t.Errorf("prometheus_listen_address without an environment value = %q, want :2112", got)
	}

	// a second call on the same command registers the flags again and must fail loudly
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Bootstrap() on the same command twice did not panic")
		}
	}()
	_ = Bootstrap(cmd)
}
