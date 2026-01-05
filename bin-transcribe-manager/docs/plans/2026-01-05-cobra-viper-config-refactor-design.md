# Cobra + Viper Configuration Refactoring Design

**Date:** 2026-01-05
**Status:** Approved
**Goal:** Refactor bin-transcribe-manager from pflag/init() pattern to modern Cobra+Viper pattern matching bin-agent-manager

## Overview

Currently, bin-transcribe-manager uses an old configuration pattern with pflag directly, global variables, and init() functions. This refactoring will modernize the configuration system to match bin-agent-manager's approach using Cobra commands, Viper for configuration management, and a structured config package.

## Current State

**Files:**
- `cmd/transcribe-manager/main.go` - Direct execution, global variables
- `cmd/transcribe-manager/init.go` - pflag-based config with repetitive bind code

**Configuration Fields:**
- `database_dsn`
- `prometheus_endpoint`, `prometheus_listen_address`
- `rabbitmq_address`
- `redis_address`, `redis_password`, `redis_database`
- `aws_access_key`, `aws_secret_key`

**Problems:**
- Repetitive flag/env binding code (156 lines for 8 configs)
- Global variables scattered across files
- init() functions run automatically, harder to test
- No clear separation of concerns

## Target State

**New Structure:**
```
internal/
  config/
    main.go          # Config package with Viper/Cobra bindings
cmd/
  transcribe-manager/
    main.go          # Cobra command structure, uses config.Get()
    (init.go DELETED)
```

**Benefits:**
- Cleaner code organization
- Consistent with agent-manager and other modern services
- Singleton pattern for thread-safe config access
- Testable configuration
- Reduced boilerplate

## Design Details

### 1. Config Package (`internal/config/main.go`)

**Config Struct:**
```go
type Config struct {
    RabbitMQAddress         string
    PrometheusEndpoint      string
    PrometheusListenAddress string
    DatabaseDSN             string
    RedisAddress            string
    RedisPassword           string
    RedisDatabase           int
    AWSAccessKey            string  // New vs agent-manager
    AWSSecretKey            string  // New vs agent-manager
}
```

**Singleton Pattern:**
- Global `globalConfig` variable
- `sync.Once` ensures config loads exactly once
- `Get() *Config` provides read-only access

**Core Functions:**

1. `Bootstrap(cmd *cobra.Command) error`
   - Called early in main() before command execution
   - Initializes logging via `initLog()`
   - Calls `bindConfig()` to set up flag/env bindings
   - Returns error if binding fails

2. `LoadGlobalConfig()`
   - Called in Cobra's `PersistentPreRunE` hook
   - Reads all values from viper into globalConfig struct
   - Uses `sync.Once` for thread safety
   - Must be called AFTER Bootstrap

3. `bindConfig(cmd *cobra.Command) error`
   - Enables `viper.AutomaticEnv()`
   - Defines all CLI flags on `cmd.PersistentFlags()`
   - Uses clean map-based binding:
   ```go
   bindings := map[string]string{
       "rabbitmq_address":          "RABBITMQ_ADDRESS",
       "prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
       // ... etc
   }
   for flagKey, envKey := range bindings {
       viper.BindPFlag(flagKey, f.Lookup(flagKey))
       viper.BindEnv(flagKey, envKey)
   }
   ```

4. `initLog()`
   - Sets up logrus with joonix formatter
   - Sets debug level
   - Private function, called only by Bootstrap

### 2. Main.go Refactoring (`cmd/transcribe-manager/main.go`)

**Main Function:**
```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "transcribe-manager",
        Short: "Voipbin Transcribe Manager Daemon",
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            config.LoadGlobalConfig()
            return nil
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            return runDaemon()
        },
    }

    if errBind := config.Bootstrap(rootCmd); errBind != nil {
        logrus.Fatalf("Failed to bootstrap config: %v", errBind)
    }

    if errExecute := rootCmd.Execute(); errExecute != nil {
        logrus.Errorf("Command execution failed: %v", errExecute)
        os.Exit(1)
    }
}
```

**Execution Flow:**
1. Create Cobra root command
2. Bootstrap config (bind flags/env)
3. Execute command
   - PersistentPreRunE: Load config into singleton
   - RunE: Call runDaemon()

**Global Variables Removed:**
Delete all package-level config variables:
- `databaseDSN`, `prometheusEndpoint`, `prometheusListenAddress`
- `rabbitMQAddress`, `redisAddress`, `redisPassword`, `redisDatabase`
- `awsAccessKey`, `awsSecretKey`

**Config Access Pattern:**
Replace all references with `config.Get()`:
```go
// Old:
sqlDB, err := commondatabasehandler.Connect(databaseDSN)

// New:
sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
```

**Functions Moved from init.go:**

1. `initSignal()` - No changes, called from runDaemon()
2. `initProm(endpoint, listen string)` - Simplified version from agent-manager (no retry loop)
3. `initCache() (cachehandler.CacheHandler, error)` - New helper function following agent-manager pattern

**runDaemon() Function:**
```go
func runDaemon() error {
    initSignal()
    initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

    log := logrus.WithField("func", "runDaemon")
    log.WithField("config", config.Get()).Info("Starting transcribe-manager...")

    sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
    if err != nil {
        return errors.Wrapf(err, "could not connect to the database")
    }
    defer commondatabasehandler.Close(sqlDB)

    cache, err := initCache()
    if err != nil {
        return errors.Wrapf(err, "could not initialize the cache")
    }

    if errRun := run(sqlDB, cache); errRun != nil {
        return errors.Wrapf(errRun, "could not run transcribe-manager")
    }

    <-chDone
    log.Info("Transcribe-manager stopped safely.")
    return nil
}
```

**Unchanged Functions:**
- `run(sqlDB, cache)` - Business logic unchanged
- `runListen()` - Unchanged
- `runSubscribe()` - Unchanged
- `runStreaming()` - Unchanged
- `signalHandler()` - Unchanged

### 3. Special Handling

**POD_IP Environment Variable:**
Remains as direct `os.Getenv("POD_IP")` call in runDaemon/run since it's read at runtime, not during config initialization. This is deployment-specific and shouldn't be in the config struct.

**Prometheus Init Simplification:**
Use agent-manager's simpler approach without retry loop. If the port is in use, fail fast with clear error.

```go
// Old approach (with retry):
for {
    err := http.ListenAndServe(listen, nil)
    if err != nil {
        logrus.Errorf("Could not start prometheus listener")
        time.Sleep(time.Second * 1)
        continue
    }
    break
}

// New approach (fail fast):
go func() {
    logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
    if err := http.ListenAndServe(listen, nil); err != nil {
        logrus.Errorf("Prometheus server error: %v", err)
    }
}()
```

## Migration Impact

### Breaking Changes
- CLI flag names remain the same format (lowercase with underscores)
- No default values (flags default to empty/"" instead of hardcoded defaults)
- Users must provide configuration explicitly

### Backward Compatibility
- **Environment variables:** Fully compatible (K8s deployments unaffected)
- **CLI usage:** Compatible (same flag names)
- **Default values:** Breaking change - no more defaults from init.go constants

### Deployment Impact
- **Kubernetes:** No impact - already uses environment variables
- **Local development:** May need to provide explicit config if previously relying on defaults

## Error Handling

1. **Bootstrap Failure:** Fatal error, immediate exit
2. **Config Load:** Protected by sync.Once, thread-safe
3. **Missing Config:** Fields default to zero values (empty string, 0)
4. **Invalid Config:** Fails at connection time with clear error
5. **Prometheus Port Conflict:** Logs error, doesn't retry

## Testing Considerations

- Config package is more testable (no init() side effects)
- Can create Config instances for testing without globals
- Bootstrap can be tested independently
- Main command flow can be tested with Cobra's testing utilities

## Implementation Checklist

- [ ] Create `internal/config/` directory
- [ ] Implement `internal/config/main.go`
- [ ] Refactor `cmd/transcribe-manager/main.go`
  - [ ] Add Cobra command structure
  - [ ] Remove global config variables
  - [ ] Replace with `config.Get()` calls
  - [ ] Move init functions from init.go
  - [ ] Implement runDaemon()
- [ ] Delete `cmd/transcribe-manager/init.go`
- [ ] Update imports (add cobra, remove unused)
- [ ] Test with environment variables
- [ ] Test with CLI flags
- [ ] Update CLAUDE.md if needed
- [ ] Update README.md example command if needed

## References

- **Template:** `/home/pchero/gitvoipbin/monorepo/bin-agent-manager/`
- **Config Package:** `bin-agent-manager/internal/config/main.go`
- **Main Pattern:** `bin-agent-manager/cmd/agent-manager/main.go`
