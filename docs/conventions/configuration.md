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
