package config

import (
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-registrar-manager/models/common"
)

// resetBootstrapState puts the process-wide state touched by Bootstrap back to its
// zero state, now and when the test ends, so the cases do not depend on the run order.
func resetBootstrapState(t *testing.T) {
	t.Helper()

	reset := func() {
		viper.Reset()
		once = sync.Once{}
		globalConfig = Config{}
		common.ResetBaseDomainNamesForTest()
	}

	formatter := logrus.StandardLogger().Formatter
	level := logrus.GetLevel()

	reset()
	t.Cleanup(func() {
		reset()
		logrus.SetFormatter(formatter)
		logrus.SetLevel(level)
	})
}

// setBootstrapEnv sets every configuration key, so values from the developer shell
// cannot change the outcome.
func setBootstrapEnv(t *testing.T, extensionDomain string) {
	t.Helper()

	t.Setenv("RABBITMQ_ADDRESS", "amqp://mq.example.test:5672")
	t.Setenv("PROMETHEUS_ENDPOINT", "/case-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":2999")
	t.Setenv("DATABASE_DSN_BIN", "bin-dsn.example.test")
	t.Setenv("DATABASE_DSN_ASTERISK", "asterisk-dsn.example.test")
	t.Setenv("REDIS_ADDRESS", "redis.example.test:6379")
	redisValue := "case-redis-value"
	t.Setenv("REDIS_PASSWORD", redisValue)
	t.Setenv("REDIS_DATABASE", "3")
	t.Setenv("DOMAIN_NAME_EXTENSION", extensionDomain)
	t.Setenv("DOMAIN_NAME_TRUNK", "trunk.example.test")
	t.Setenv("DOMAIN_SHORT_LABEL_ENABLED", "true")
}

func Test_Bootstrap_loadsEnvironment(t *testing.T) {
	resetBootstrapState(t)
	setBootstrapEnv(t, "ext.example.test")

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	cfg := Get()
	if cfg.RabbitMQAddress != "amqp://mq.example.test:5672" || cfg.PrometheusEndpoint != "/case-metrics" || cfg.PrometheusListenAddress != ":2999" {
		t.Errorf("Wrong match. broker or metrics settings: %+v", cfg)
	}
	if cfg.DatabaseDSNBin != "bin-dsn.example.test" || cfg.DatabaseDSNAsterisk != "asterisk-dsn.example.test" {
		t.Errorf("Wrong match. database settings: %+v", cfg)
	}
	if cfg.RedisAddress != "redis.example.test:6379" || cfg.RedisPassword != "case-redis-value" || cfg.RedisDatabase != 3 {
		t.Errorf("Wrong match. redis settings: %+v", cfg)
	}
	if cfg.DomainNameExtension != "ext.example.test" || cfg.DomainNameTrunk != "trunk.example.test" || !cfg.DomainShortLabelEnabled {
		t.Errorf("Wrong match. domain settings: %+v", cfg)
	}

	// the base domain names are registered for realm generation
	if res := common.BaseDomainNameExtension(); res != "ext.example.test" {
		t.Errorf("Wrong match. expect: ext.example.test, got: %s", res)
	}
	if res := common.GenerateRealmTrunkDomain("case"); res != "case.trunk.example.test" {
		t.Errorf("Wrong match. expect: case.trunk.example.test, got: %s", res)
	}
}

func Test_Bootstrap_invalidBaseDomain(t *testing.T) {
	resetBootstrapState(t)
	setBootstrapEnv(t, "not a valid domain")

	err := Bootstrap(&cobra.Command{})
	if err == nil || !strings.Contains(err.Error(), "could not set base domain names") {
		t.Fatalf("Wrong match. expect: the base domain error, got: %v", err)
	}
}

func Test_Bootstrap_secondCallIsRejected(t *testing.T) {
	resetBootstrapState(t)
	setBootstrapEnv(t, "ext.example.test")

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// a second call (new command, so the flags can be defined again) must not change
	// the base domain names
	t.Setenv("DOMAIN_NAME_EXTENSION", "other.example.test")
	err := Bootstrap(&cobra.Command{})
	if err == nil || !strings.Contains(err.Error(), "already been initialized") {
		t.Fatalf("Wrong match. expect: the already initialized error, got: %v", err)
	}
	if res := common.BaseDomainNameExtension(); res != "ext.example.test" {
		t.Errorf("Wrong match. expect: the first base domain to be kept, got: %s", res)
	}
	if res := Get().DomainNameExtension; res != "ext.example.test" {
		t.Errorf("Wrong match. expect: the loaded configuration to be kept, got: %s", res)
	}
}
