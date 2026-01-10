# Cobra/Viper Configuration Standardization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate all 25+ services from legacy `init.go` pattern to modern Cobra/Viper configuration pattern.

**Architecture:** Each service gets an `internal/config` package with a singleton Config struct, Bootstrap function for Cobra integration, and helper functions for logging/Prometheus. Main function transforms to use Cobra command structure with explicit initialization order.

**Tech Stack:** Go, Cobra (CLI framework), Viper (configuration management), Logrus (logging), Prometheus (metrics)

**Reference Implementation:** `bin-flow-manager/internal/config/main.go`

**Services to Migrate (alphabetical order):**
1. bin-ai-manager
2. bin-api-manager
3. bin-billing-manager
4. bin-call-manager
5. bin-campaign-manager
6. bin-chat-manager
7. bin-conference-manager
8. bin-conversation-manager
9. bin-dbscheme-manager
10. bin-email-manager
11. bin-hook-manager
12. bin-message-manager
13. bin-openapi-manager
14. bin-outdial-manager
15. bin-pipecat-manager
16. bin-queue-manager
17. bin-registrar-manager
18. bin-route-manager
19. bin-sentinel-manager
20. bin-storage-manager
21. bin-tag-manager
22. bin-transcribe-manager
23. bin-transfer-manager
24. bin-tts-manager
25. bin-webhook-manager

---

## Migration Pattern (applies to each service)

Each service migration follows these steps:
1. Analyze existing `init.go` for configuration fields
2. Create `internal/config/main.go` with Config struct
3. Transform `cmd/<service>/main.go` to use Cobra
4. Update all references from global vars to `config.Get()`
5. Delete `cmd/<service>/init.go`
6. Run verification workflow
7. Commit if tests pass

---

## Task 1: Migrate bin-ai-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-ai-manager/cmd/ai-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-ai-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-ai-manager/cmd/ai-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-ai-manager/cmd/ai-manager/init.go`

**Step 1: Analyze existing init.go**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-ai-manager
cat cmd/ai-manager/init.go | grep "pflag\." | head -20
```

Expected: List of all pflag declarations showing configuration fields

**Step 2: Create internal/config directory**

```bash
mkdir -p internal/config
```

**Step 3: Create internal/config/main.go**

Copy template from flow-manager and customize with service-specific fields:

```go
package config

import (
	"net/http"
	"sync"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds process-wide configuration values loaded from command-line
// flags and environment variables for the service.
type Config struct {
	RabbitMQAddress         string // RabbitMQ server address
	PrometheusEndpoint      string // Prometheus metrics endpoint
	PrometheusListenAddress string // Prometheus listen address
	DatabaseDSN             string // Database connection DSN
	RedisAddress            string // Redis server address
	RedisPassword           string // Redis password
	RedisDatabase           int    // Redis database index
	// Add service-specific fields discovered from init.go analysis
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}
	initProm()
	return nil
}

func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")
	// Add service-specific flags from init.go

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		// Add service-specific env bindings
	}

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

func Get() *Config {
	return &globalConfig
}

func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDatabase:           viper.GetInt("redis_database"),
			// Add service-specific field loading
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

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

**Step 4: Read current main.go to understand structure**

```bash
head -100 cmd/ai-manager/main.go
```

**Step 5: Transform cmd/ai-manager/main.go**

Add imports, Cobra structure, and replace global variable access:

```go
package main

import (
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-ai-manager/internal/config"
	// ... other existing imports
)

const serviceName = commonoutline.ServiceNameAIManager

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ai-manager",
		Short: "Voipbin AI Manager Daemon",
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
	log.WithField("config", config.Get()).Info("Starting ai-manager...")

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	defer commondatabasehandler.Close(sqlDB)

	cache, err := initCache()
	if err != nil {
		return errors.Wrapf(err, "could not initialize the cache")
	}

	if errStart := runServices(sqlDB, cache); errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	log.Info("AI-manager stopped safely.")
	return nil
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		logrus.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

// Keep existing initCache, runServices, etc. functions
// Replace all global variable references with config.Get().FieldName
```

**Step 6: Find and replace all global variable usage**

```bash
# Search for uses of old global variables
grep -r "databaseDSN" --include="*.go" . | grep -v "init.go" | grep -v "vendor"
grep -r "rabbitMQAddress" --include="*.go" . | grep -v "init.go" | grep -v "vendor"
grep -r "redisAddress" --include="*.go" . | grep -v "init.go" | grep -v "vendor"
```

Replace each occurrence with `config.Get().FieldName`

**Step 7: Delete init.go**

```bash
rm cmd/ai-manager/init.go
```

**Step 8: Run verification workflow**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-ai-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All tests pass, no linting errors

**Step 9: Fix any compilation or test errors**

Common issues:
- Missing imports for config package
- Incorrect field names (check capitalization)
- Type mismatches for array fields
- Config accessed before LoadGlobalConfig

**Step 10: Commit if tests pass**

```bash
git add -A
git commit -m "$(cat <<'EOF'
VOIP-1188: Refactor ai-manager config to Cobra/Viper

Migrate bin-ai-manager from legacy init.go pattern to modern Cobra/Viper
configuration management.

Changes:
- Create internal/config package with Config struct
- Transform main.go to use Cobra command structure
- Replace global variables with config.Get() accessor
- Move Prometheus initialization to config.Bootstrap()
- Delete init.go file

Tests: All passing
EOF
)"
```

---

## Task 2: Migrate bin-api-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-api-manager/cmd/api-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-api-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-api-manager/cmd/api-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-api-manager/cmd/api-manager/init.go`

**Note:** bin-api-manager likely has additional configuration for JWT, CORS, ZMQ, and Swagger.

**Step 1: Analyze existing init.go**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager
cat cmd/api-manager/init.go | grep -E "(pflag\.|const default)" | head -30
```

**Step 2-10: Follow same pattern as Task 1**

Pay special attention to:
- JWT configuration fields
- CORS origins (likely []string type)
- ZMQ address
- Swagger base path
- Any API-specific settings

```bash
# Verification
cd /home/pchero/gitvoipbin/monorepo/bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

```bash
# Commit
git add -A
git commit -m "VOIP-1188: Refactor api-manager config to Cobra/Viper"
```

---

## Task 3: Migrate bin-billing-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-billing-manager/cmd/billing-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-billing-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-billing-manager/cmd/billing-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-billing-manager/cmd/billing-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-billing-manager
# Analyze, create config, transform main, delete init, verify, commit
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor billing-manager config to Cobra/Viper"
```

---

## Task 4: Migrate bin-call-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-call-manager/cmd/call-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-call-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-call-manager/cmd/call-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-call-manager/cmd/call-manager/init.go`

**Step 1: Analyze existing init.go**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-call-manager
cat cmd/call-manager/init.go | grep -E "(pflag\.|const default)" | head -30
```

Expected: Homer-specific configuration (HomerAPIAddress, HomerAuthToken, HomerWhitelist)

**Step 2: Create internal/config/main.go**

Add Homer-specific fields to Config struct:

```go
type Config struct {
	RabbitMQAddress         string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	DatabaseDSN             string
	RedisAddress            string
	RedisPassword           string
	RedisDatabase           int
	HomerAPIAddress         string   // Homer SIP capture API address
	HomerAuthToken          string   // Homer authentication token
	HomerWhitelist          []string // IP whitelist for Homer
}
```

**Step 3: Add Homer bindings in bindConfig**

```go
f.String("homer_api_address", "", "Homer API server address")
f.String("homer_auth_token", "", "Homer authentication token")
f.String("homer_whitelist", "", "Comma-separated IP whitelist for Homer")

bindings["homer_api_address"] = "HOMER_API_ADDRESS"
bindings["homer_auth_token"] = "HOMER_AUTH_TOKEN"
bindings["homer_whitelist"] = "HOMER_WHITELIST"
```

**Step 4: Add Homer field loading in LoadGlobalConfig**

```go
homerWhitelistStr := viper.GetString("homer_whitelist")
var homerWhitelist []string
if homerWhitelistStr != "" {
	homerWhitelist = strings.Split(homerWhitelistStr, ",")
}

globalConfig = Config{
	// ... common fields
	HomerAPIAddress: viper.GetString("homer_api_address"),
	HomerAuthToken:  viper.GetString("homer_auth_token"),
	HomerWhitelist:  homerWhitelist,
}
```

**Steps 5-10: Follow same pattern as Task 1**

```bash
# Verification
cd /home/pchero/gitvoipbin/monorepo/bin-call-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

```bash
# Commit
git add -A
git commit -m "VOIP-1188: Refactor call-manager config to Cobra/Viper"
```

---

## Task 5: Migrate bin-campaign-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-campaign-manager/cmd/campaign-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-campaign-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-campaign-manager/cmd/campaign-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-campaign-manager/cmd/campaign-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-campaign-manager
# Analyze, create config, transform main, delete init, verify, commit
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor campaign-manager config to Cobra/Viper"
```

---

## Task 6: Migrate bin-chat-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-chat-manager/cmd/chat-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-chat-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-chat-manager/cmd/chat-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-chat-manager/cmd/chat-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-chat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor chat-manager config to Cobra/Viper"
```

---

## Task 7: Migrate bin-conference-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-conference-manager/cmd/conference-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-conference-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-conference-manager/cmd/conference-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-conference-manager/cmd/conference-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-conference-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor conference-manager config to Cobra/Viper"
```

---

## Task 8: Migrate bin-conversation-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-conversation-manager/cmd/conversation-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-conversation-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-conversation-manager/cmd/conversation-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-conversation-manager/cmd/conversation-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-conversation-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor conversation-manager config to Cobra/Viper"
```

---

## Task 9: Migrate bin-dbscheme-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager/cmd/dbscheme-manager/init.go` (if exists)
- Create: `/home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager/cmd/dbscheme-manager/main.go`

**Note:** This service may not have init.go. Check first.

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-dbscheme-manager
ls -la cmd/dbscheme-manager/
```

If no init.go exists, skip this task.

If init.go exists, follow Task 1 pattern.

---

## Task 10: Migrate bin-email-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-email-manager/cmd/email-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-email-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-email-manager/cmd/email-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-email-manager/cmd/email-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-email-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor email-manager config to Cobra/Viper"
```

---

## Task 11: Migrate bin-hook-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-hook-manager/cmd/hook-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-hook-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-hook-manager/cmd/hook-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-hook-manager/cmd/hook-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-hook-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor hook-manager config to Cobra/Viper"
```

---

## Task 12: Migrate bin-message-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-message-manager/cmd/message-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-message-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-message-manager/cmd/message-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-message-manager/cmd/message-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-message-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor message-manager config to Cobra/Viper"
```

---

## Task 13: Migrate bin-openapi-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-openapi-manager/cmd/openapi-manager/` (check for init.go)
- Create: `/home/pchero/gitvoipbin/monorepo/bin-openapi-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-openapi-manager/cmd/openapi-manager/main.go`

**Note:** This service may not have init.go. Check first.

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-openapi-manager
ls -la cmd/openapi-manager/
```

If no init.go exists, skip this task. If exists, follow Task 1 pattern.

---

## Task 14: Migrate bin-outdial-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-outdial-manager/cmd/outdial-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-outdial-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-outdial-manager/cmd/outdial-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-outdial-manager/cmd/outdial-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-outdial-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor outdial-manager config to Cobra/Viper"
```

---

## Task 15: Migrate bin-pipecat-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-pipecat-manager/cmd/pipecat-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-pipecat-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-pipecat-manager/cmd/pipecat-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-pipecat-manager/cmd/pipecat-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor pipecat-manager config to Cobra/Viper"
```

---

## Task 16: Migrate bin-queue-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-queue-manager/cmd/queue-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-queue-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-queue-manager/cmd/queue-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-queue-manager/cmd/queue-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor queue-manager config to Cobra/Viper"
```

---

## Task 17: Migrate bin-registrar-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-registrar-manager/cmd/registrar-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-registrar-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-registrar-manager/cmd/registrar-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-registrar-manager/cmd/registrar-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-registrar-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor registrar-manager config to Cobra/Viper"
```

---

## Task 18: Migrate bin-route-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-route-manager/cmd/route-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-route-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-route-manager/cmd/route-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-route-manager/cmd/route-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-route-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor route-manager config to Cobra/Viper"
```

---

## Task 19: Migrate bin-sentinel-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-sentinel-manager/cmd/sentinel-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-sentinel-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-sentinel-manager/cmd/sentinel-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-sentinel-manager/cmd/sentinel-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-sentinel-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor sentinel-manager config to Cobra/Viper"
```

---

## Task 20: Migrate bin-storage-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-storage-manager/cmd/storage-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-storage-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-storage-manager/cmd/storage-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-storage-manager/cmd/storage-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-storage-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor storage-manager config to Cobra/Viper"
```

---

## Task 21: Migrate bin-tag-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-tag-manager/cmd/tag-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-tag-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-tag-manager/cmd/tag-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-tag-manager/cmd/tag-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-tag-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor tag-manager config to Cobra/Viper"
```

---

## Task 22: Migrate bin-transcribe-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-transcribe-manager/cmd/transcribe-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-transcribe-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-transcribe-manager/cmd/transcribe-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-transcribe-manager/cmd/transcribe-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-transcribe-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor transcribe-manager config to Cobra/Viper"
```

---

## Task 23: Migrate bin-transfer-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-transfer-manager/cmd/transfer-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-transfer-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-transfer-manager/cmd/transfer-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-transfer-manager/cmd/transfer-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-transfer-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor transfer-manager config to Cobra/Viper"
```

---

## Task 24: Migrate bin-tts-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-tts-manager/cmd/tts-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-tts-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-tts-manager/cmd/tts-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-tts-manager/cmd/tts-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-tts-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor tts-manager config to Cobra/Viper"
```

---

## Task 25: Migrate bin-webhook-manager

**Files:**
- Read: `/home/pchero/gitvoipbin/monorepo/bin-webhook-manager/cmd/webhook-manager/init.go`
- Create: `/home/pchero/gitvoipbin/monorepo/bin-webhook-manager/internal/config/main.go`
- Modify: `/home/pchero/gitvoipbin/monorepo/bin-webhook-manager/cmd/webhook-manager/main.go`
- Delete: `/home/pchero/gitvoipbin/monorepo/bin-webhook-manager/cmd/webhook-manager/init.go`

**Steps 1-10: Follow same pattern as Task 1**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-webhook-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add -A && git commit -m "VOIP-1188: Refactor webhook-manager config to Cobra/Viper"
```

---

## Task 26: Final Verification

**Step 1: Run comprehensive test across all services**

```bash
cd /home/pchero/gitvoipbin/monorepo
find . -maxdepth 2 -name "go.mod" -execdir bash -c "echo 'Testing $(pwd)' && go mod tidy && go mod vendor && go generate ./... && go test ./..." \;
```

Expected: All services pass tests

**Step 2: Verify no init.go files remain**

```bash
cd /home/pchero/gitvoipbin/monorepo
find . -path "*/cmd/*/init.go" -type f
```

Expected: Empty output (no init.go files found)

**Step 3: Verify all services have internal/config**

```bash
cd /home/pchero/gitvoipbin/monorepo
for dir in bin-*-manager; do
  if [ -d "$dir/internal/config" ]; then
    echo "✅ $dir has internal/config"
  else
    echo "❌ $dir MISSING internal/config"
  fi
done
```

Expected: All services show ✅

**Step 4: Test help output on a few services**

```bash
cd /home/pchero/gitvoipbin/monorepo/bin-call-manager
go build ./cmd/...
./call-manager --help
```

Expected: Cobra-generated help showing all flags

**Step 5: Create final summary commit**

```bash
cd /home/pchero/gitvoipbin/monorepo
git add -A
git commit -m "$(cat <<'EOF'
VOIP-1188: Complete Cobra/Viper migration for all services

Successfully migrated all 25+ services from legacy init.go pattern to
modern Cobra/Viper configuration management.

Summary:
- 25+ services now use internal/config package
- All services use Cobra command structure
- Zero init.go files remaining
- All tests passing
- Consistent configuration pattern across monorepo

Benefits:
- Automatic --help support for all services
- Better testability (no global state in init())
- Easier to maintain and extend configuration
- Consistent developer experience across services
EOF
)"
```

---

## Troubleshooting Guide

### Issue: Compilation error "undefined: config"

**Cause:** Missing import for config package

**Fix:**
```go
import "monorepo/bin-<service>/internal/config"
```

### Issue: Nil pointer dereference in config.Get()

**Cause:** Accessing config before LoadGlobalConfig() called

**Fix:** Ensure LoadGlobalConfig() is in PersistentPreRunE, called before runDaemon()

### Issue: Tests fail with "flag already defined"

**Cause:** Cobra command created multiple times in tests

**Fix:** Ensure tests don't call main() or Bootstrap() multiple times

### Issue: Type mismatch for []string fields

**Cause:** Old code used string, new code uses []string

**Fix:** Update Config struct and handle splitting in LoadGlobalConfig():
```go
whitelistStr := viper.GetString("whitelist")
var whitelist []string
if whitelistStr != "" {
    whitelist = strings.Split(whitelistStr, ",")
}
```

### Issue: golangci-lint errors about unused imports

**Cause:** Removed global variables but imports still present

**Fix:** Run `goimports` or manually remove unused imports

### Issue: Prometheus not starting

**Cause:** initProm() called before config loaded

**Fix:** Ensure initProm() called in config.Bootstrap() after bindConfig()

---

## Success Criteria

- ✅ All 25+ services migrated to Cobra/Viper pattern
- ✅ Zero init.go files remaining
- ✅ All services pass verification workflow
- ✅ All tests passing
- ✅ All services have internal/config package
- ✅ All services support --help flag
- ✅ Consistent configuration pattern across monorepo
