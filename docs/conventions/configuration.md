# Configuration

### 12.1 Cobra + Viper + sync.Once

Every service uses the same configuration pattern:

```go
// CORRECT — internal/config/main.go
package config

var (
    globalConfig Config
    once         sync.Once
)

type Config struct {
    RabbitMQAddress         string
    DatabaseDSN             string
    RedisAddress            string
    RedisPassword           string
    RedisDatabase           int
    PrometheusEndpoint      string
    PrometheusListenAddress string
    // service-specific fields...
}

func Bootstrap(cmd *cobra.Command) error {
    initLog()
    return bindConfig(cmd)
}

func LoadGlobalConfig() {
    once.Do(func() {
        globalConfig = Config{
            DatabaseDSN: viper.GetString("database_dsn"),
            // ...
        }
    })
}

func Get() *Config { return &globalConfig }
```

All 33 service `internal/config` packages follow this shape; after
`NOJIRA-Fix-sentinel-config-once-retry-bug`, 32 of them still wrap the loader in `sync.Once` and
`bin-sentinel-manager` is the sole exception, on two linked axes. Its `LoadGlobalConfig` returns
an `error` (the canonical form above returns nothing) so the CLI can propagate a validation
failure; and *because* it can now fail, it deliberately does **not** wrap the load in `once` — a
`once`-wrapped loader swallows every call after the first, so the retry after a validation failure
would silently do nothing. See `bin-sentinel-manager/internal/config/config.go` for the
unconditional load-validate-store alternative.

### 12.2 Environment Variable Binding

Each config field maps to a CLI flag and an environment variable:

```go
// CORRECT
f := cmd.PersistentFlags()
f.String("database_dsn", "", "Database connection string")
viper.BindPFlag("database_dsn", f.Lookup("database_dsn"))
viper.BindEnv("database_dsn", "DATABASE_DSN")
```

### 12.3 Logging Initialization

All services set logrus to debug level with joonix formatter:

```go
// CORRECT
func initLog() {
    logrus.SetFormatter(joonix.NewFormatter())
    logrus.SetLevel(logrus.DebugLevel)
}
```

---
