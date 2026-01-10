# Configuration Standardization: Migrate All Services to Cobra/Viper Pattern

**Date:** 2026-01-10
**Status:** Approved
**Scope:** All 25+ manager services in monorepo (excluding bin-flow-manager, bin-agent-manager, bin-customer-manager, bin-number-manager which already use this pattern)

## Overview

This design standardizes configuration management across all services in the VoIPbin monorepo by migrating from the legacy `init.go` pattern to the modern Cobra/Viper pattern established in `bin-flow-manager`.

## Current State

### Services Already Modernized (4 services)
- `bin-flow-manager` ✅ (reference implementation)
- `bin-agent-manager` ✅
- `bin-customer-manager` ✅
- `bin-number-manager` ✅

### Services Requiring Migration (25+ services)
Services still using the old pattern with `init.go`:
- Core: `bin-call-manager`, `bin-conference-manager`, `bin-api-manager`
- AI/Messaging: `bin-ai-manager`, `bin-message-manager`, `bin-email-manager`, `bin-chat-manager`
- Queue/Routing: `bin-queue-manager`, `bin-route-manager`, `bin-transfer-manager`
- Campaign: `bin-campaign-manager`, `bin-outdial-manager`
- Infrastructure: `bin-webhook-manager`, `bin-hook-manager`, `bin-storage-manager`, `bin-sentinel-manager`, `bin-billing-manager`
- Communication: `bin-conversation-manager`, `bin-transcribe-manager`, `bin-tts-manager`, `bin-pipecat-manager`
- Other: `bin-registrar-manager`, `bin-dbscheme-manager`, `bin-openapi-manager`

### Problems with Legacy Pattern

**`init.go` file issues:**
- 100+ lines of repetitive Viper binding code
- Global variables scattered in `main.go` (e.g., `var databaseDSN = ""`)
- Configuration, logging, signal handling, and Prometheus initialization all mixed in `init()` function
- Poor testability due to global state initialized in `init()`
- Inconsistent configuration access patterns
- No `--help` support for discovering available flags

**Maintenance burden:**
- Adding new common configuration requires updating 25+ services individually
- No standardization makes onboarding difficult
- Hard to validate configuration completeness

## Design Goals

1. **Standardization**: All services follow identical configuration pattern
2. **Maintainability**: Easy to add new common configuration fields
3. **Testability**: No global state in `init()`, explicit initialization order
4. **Discoverability**: Cobra provides automatic `--help` support
5. **Flexibility**: Each service can extend base configuration with service-specific fields
6. **Backward Compatibility**: Preserve all existing configuration flags and environment variables

## Architecture

### 1. Configuration Package Structure

Each service will have `internal/config/main.go` with:

#### Config Struct
```go
type Config struct {
    // Common fields (all services)
    RabbitMQAddress         string
    DatabaseDSN             string
    RedisAddress            string
    RedisPassword           string
    RedisDatabase           int
    PrometheusEndpoint      string
    PrometheusListenAddress string

    // Service-specific fields (example from call-manager)
    HomerAPIAddress string
    HomerAuthToken  string
    HomerWhitelist  []string
}
```

**Design decisions:**
- Single struct per service containing all configuration
- Common fields present in all services
- Service-specific fields added as needed
- Descriptive comments on all fields
- Go naming conventions (CamelCase for exported fields)

#### Bootstrap Function
```go
func Bootstrap(cmd *cobra.Command) error {
    initLog()
    if errBind := bindConfig(cmd); errBind != nil {
        return errors.Wrapf(errBind, "could not bind config")
    }
    initProm()
    return nil
}
```

**Responsibilities:**
- Initialize logging (logrus with joonix formatter)
- Bind CLI flags and environment variables
- Initialize Prometheus metrics server
- Return error if binding fails

#### Configuration Access
```go
var (
    globalConfig Config
    once         sync.Once
)

func Get() *Config {
    return &globalConfig
}

func LoadGlobalConfig() {
    once.Do(func() {
        globalConfig = Config{
            RabbitMQAddress: viper.GetString("rabbitmq_address"),
            DatabaseDSN: viper.GetString("database_dsn"),
            // ... all fields
        }
    })
}
```

**Design decisions:**
- Thread-safe singleton pattern using `sync.Once`
- Configuration loaded exactly once in Cobra's `PersistentPreRunE`
- Immutable after loading (no setters)
- Clean access via `config.Get().FieldName`

#### Helper Functions

**bindConfig:**
```go
func bindConfig(cmd *cobra.Command) error {
    viper.AutomaticEnv()
    f := cmd.PersistentFlags()

    // Define all flags
    f.String("rabbitmq_address", "", "RabbitMQ server address")
    f.String("database_dsn", "", "Database connection DSN")
    // ... all flags

    // Map flags to environment variables
    bindings := map[string]string{
        "rabbitmq_address": "RABBITMQ_ADDRESS",
        "database_dsn": "DATABASE_DSN",
        // ... all bindings
    }

    // Bind each flag and env var
    for flagKey, envKey := range bindings {
        if errBind := viper.BindPFlag(flagKey, f.Lookup(flagKey)); errBind != nil {
            return errors.Wrapf(errBind, "could not bind flag. key: %s", flagKey)
        }
        if errBind := viper.BindEnv(flagKey, envKey); errBind != nil {
            return errors.Wrapf(errBind, "could not bind the env. key: %s", envKey)
        }
    }

    return nil
}
```

**initLog:**
```go
func initLog() {
    logrus.SetFormatter(joonix.NewFormatter())
    logrus.SetLevel(logrus.DebugLevel)
}
```

**initProm:**
```go
func initProm() {
    cfg := Get()
    if cfg.PrometheusEndpoint == "" || cfg.PrometheusListenAddress == "" {
        logrus.Debug("Prometheus metrics server disabled")
        return
    }

    http.Handle(cfg.PrometheusEndpoint, promhttp.Handler())
    go func() {
        logrus.Infof("Prometheus metrics server starting on %s%s",
            cfg.PrometheusListenAddress, cfg.PrometheusEndpoint)
        if err := http.ListenAndServe(cfg.PrometheusListenAddress, nil); err != nil {
            logrus.Errorf("Prometheus server error: %v", err)
        }
    }()
}
```

### 2. Main Function Transformation

Each service's `cmd/<service-name>/main.go` will be transformed:

#### Before (Legacy Pattern)
```go
var (
    databaseDSN = ""
    rabbitMQAddress = ""
    // ... many global variables
)

func main() {
    // Configuration already initialized in init()
    sqlDB, err := commondatabasehandler.Connect(databaseDSN)
    // ...
}
```

#### After (Modern Pattern)
```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "service-name",
        Short: "Voipbin Service Name Daemon",
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

func runDaemon() error {
    initSignal()

    log := logrus.WithField("func", "runDaemon")
    log.WithField("config", config.Get()).Info("Starting service-name...")

    sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
    // ... rest of initialization using config.Get()

    <-chDone
    log.Info("Service stopped safely.")
    return nil
}
```

**Key Changes:**
- **Remove**: `init.go` file entirely
- **Remove**: Global configuration variables
- **Add**: Cobra command structure
- **Add**: Import `"monorepo/bin-<service>/internal/config"`
- **Move**: Signal handling from `init()` to `runDaemon()`
- **Move**: Prometheus from `init()` to `config.Bootstrap()`
- **Change**: All `databaseDSN` → `config.Get().DatabaseDSN`

### 3. Service-Specific Configuration Handling

#### Standard Fields (All Services)
```go
DatabaseDSN             string // MySQL connection string
RabbitMQAddress         string // RabbitMQ server address
RedisAddress            string // Redis server address
RedisPassword           string // Redis password (optional)
RedisDatabase           int    // Redis database index
PrometheusEndpoint      string // Prometheus metrics path (e.g., "/metrics")
PrometheusListenAddress string // Prometheus listen address (e.g., ":2112")
```

#### Service-Specific Examples

**bin-call-manager:**
```go
HomerAPIAddress string   // Homer SIP capture API address
HomerAuthToken  string   // Homer authentication token
HomerWhitelist  []string // IP whitelist for Homer (comma-separated)
```

**bin-api-manager (likely):**
```go
JWTSecret       string   // JWT signing secret
CORSOrigins     []string // Allowed CORS origins
ZMQAddress      string   // ZeroMQ address
SwaggerBasePath string   // Swagger UI base path
```

#### Discovery Process

For each service:
1. Read existing `init.go` to find all `pflag` declarations
2. Extract default values from constants (e.g., `defaultHomerAPIAddress`)
3. Map snake_case flags to CamelCase struct fields
4. Handle special types (comma-separated lists → `[]string`)
5. Preserve all flag descriptions

## Migration Workflow

### Per-Service Steps

**Step 1: Create `internal/config/main.go`**
- Copy `bin-flow-manager/internal/config/main.go` as template
- Update `Config` struct with service-specific fields from old `init.go`
- Update `bindConfig()` to include all flags and env bindings
- Preserve default values from old constants

**Step 2: Transform `cmd/<service>/main.go`**
- Add Cobra command structure
- Add config package import
- Replace global variables with `config.Get()` calls
- Move signal handling to `runDaemon()`
- Remove Prometheus initialization (now in config.Bootstrap)

**Step 3: Update Package References**
- Search codebase for old global variable usage
- Replace with `config.Get().FieldName`
- Common locations: handler constructors, database connections, cache init

**Step 4: Delete `cmd/<service>/init.go`**
- After verifying all configuration migrated

**Step 5: Immediate Verification**
```bash
cd bin-<service>
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 6: Fix Issues Before Next Service**
- Do not proceed until current service passes all tests

### Service Migration Order

Process services alphabetically for consistency:
1. bin-ai-manager
2. bin-api-manager
3. bin-billing-manager
4. bin-call-manager
5. bin-campaign-manager
6. ... (continue alphabetically)

Verify each service individually before moving to next.

## Testing Strategy

### Per-Service Validation

After each service migration, run mandatory verification:
```bash
cd bin-<service-name>
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**What this catches:**
- Import errors from config package
- Missing configuration fields
- Broken tests due to configuration changes
- Type mismatches
- Linting issues

### Common Issues to Watch

1. **Type mismatches**: Old code using `string` for `[]string` fields
2. **Default values**: Ensure defaults preserved from old constants
3. **Validation logic**: Move any validation from `init()` to appropriate location
4. **Hidden global usage**: Variables used deep in handler packages
5. **Nil pointer dereferences**: If config accessed before `LoadGlobalConfig()`
6. **Flag name mismatches**: Ensure exact flag names preserved for backward compatibility

### Final Validation

After all services migrated:
```bash
# From monorepo root - verify ALL services
find . -maxdepth 2 -name "go.mod" -execdir bash -c "go mod tidy && go mod vendor && go generate ./... && go test ./..." \;
```

## Rollout Strategy

### Git Workflow

**Branch Creation:**
Ask user for confirmation before creating branch.

Suggested branch name: `NOJIRA-Standardize_config_cobra_viper_all_services`

**Commit Strategy:**

Option A: One commit per service
- Format: `NOJIRA-Refactor_<service>_config_to_cobra_viper`
- Allows bisecting if issues arise

Option B: Batch commits (5-7 services each)
- Format: `NOJIRA-Refactor_config_batch_<N>_cobra_viper`
- Fewer commits, still manageable

**Verification Before Commit:**
Each service MUST pass verification workflow before committing.

### Timeline Expectations

- **Per service**: 15-30 minutes (most are straightforward)
- **Complex services** (api-manager, call-manager): 45-60 minutes
- **Total effort**: 8-12 hours for all services
- **Testing**: Additional 2-3 hours for comprehensive validation

## Benefits

### Immediate Benefits

1. **Consistency**: All services use identical configuration pattern
2. **Discoverability**: `./service-name --help` shows all available flags
3. **Documentation**: Config struct comments document all fields
4. **Testability**: No global state in `init()`, explicit initialization
5. **Maintainability**: Easy to add new common fields across all services

### Long-term Benefits

1. **Onboarding**: New developers see consistent pattern across all services
2. **Configuration management**: Easier to validate config completeness
3. **Future refactoring**: Cleaner architecture enables further improvements
4. **Error handling**: Explicit error returns from config initialization
5. **Flexibility**: Services can easily add new configuration fields

## Risks and Mitigations

### Risk: Breaking existing deployments

**Mitigation:**
- Preserve exact flag names and environment variable names
- Preserve default values
- Backward compatibility tested via verification workflow

### Risk: Accumulating errors across services

**Mitigation:**
- Verify each service individually before moving to next
- Fix issues immediately, don't batch
- Early services validate the pattern works

### Risk: Service-specific edge cases

**Mitigation:**
- Carefully review each `init.go` for special logic
- Handle special types (arrays, custom parsing)
- Test thoroughly per service

### Risk: Time investment

**Mitigation:**
- Pattern is proven (4 services already use it)
- Most services straightforward (only common config)
- Benefits outweigh one-time migration cost

## Success Criteria

1. ✅ All 25+ services migrated to Cobra/Viper pattern
2. ✅ Zero `init.go` files remaining in any service
3. ✅ All services pass verification workflow
4. ✅ All existing flags and env vars preserved
5. ✅ Consistent `internal/config` structure across all services
6. ✅ Documentation updated (CLAUDE.md if needed)

## References

- **Reference Implementation**: `bin-flow-manager/internal/config/main.go`
- **Cobra Documentation**: https://github.com/spf13/cobra
- **Viper Documentation**: https://github.com/spf13/viper
- **Monorepo CLAUDE.md**: Verification workflow requirements

## Appendix: Example Migration

### Before: bin-call-manager

**init.go (210 lines):**
```go
var (
    databaseDSN = ""
    rabbitMQAddress = ""
    // ... many globals
)

func init() {
    initVariable()
    initLog()
    initSignal()
    initProm(prometheusEndpoint, prometheusListenAddress)
}

func initVariable() {
    // 180 lines of repetitive binding code
}
```

**main.go:**
```go
func main() {
    log := logrus.WithField("func", "main")
    sqlDB, err := commondatabasehandler.Connect(databaseDSN)
    // ...
}
```

### After: bin-call-manager

**internal/config/main.go (120 lines):**
```go
type Config struct {
    RabbitMQAddress         string
    DatabaseDSN             string
    RedisAddress            string
    RedisPassword           string
    RedisDatabase           int
    PrometheusEndpoint      string
    PrometheusListenAddress string
    HomerAPIAddress         string
    HomerAuthToken          string
    HomerWhitelist          []string
}

func Bootstrap(cmd *cobra.Command) error { ... }
func bindConfig(cmd *cobra.Command) error { ... }
func Get() *Config { ... }
func LoadGlobalConfig() { ... }
```

**cmd/call-manager/main.go (140 lines):**
```go
import "monorepo/bin-call-manager/internal/config"

func main() {
    rootCmd := &cobra.Command{
        Use: "call-manager",
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

func runDaemon() error {
    initSignal()
    log := logrus.WithField("func", "runDaemon")

    sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
    // ... rest using config.Get()
}
```

**Result:**
- Removed 210-line `init.go`
- Cleaner, more testable code
- Automatic `--help` support
- Consistent with other services
