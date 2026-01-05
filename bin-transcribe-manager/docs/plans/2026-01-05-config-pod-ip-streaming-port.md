# Configuration Refactoring: POD_IP and Streaming Listen Port

**Date:** 2026-01-05
**Status:** Approved

## Overview and Goals

This refactoring moves the `POD_IP` environment variable handling into the centralized `internal/config` package and introduces a new configurable streaming listen port. Currently, POD_IP is read directly via `os.Getenv()` in `main.go`, and the port is hardcoded to `8080`. This change will:

1. **Centralize configuration**: Move POD_IP into the Config struct alongside other environment variables
2. **Add port configurability**: Introduce `StreamingListenPort` with a default value of `8080`
3. **Maintain validation semantics**: Keep runtime validation in `main.go` for POD_IP (fail if empty)
4. **Preserve backward compatibility**: Existing Kubernetes deployments continue to work without changes

**Configuration Fields to Add:**
- `PodIP string`: The IP address for AudioSocket streaming listener (from `POD_IP` env var)
- `StreamingListenPort int`: The port for AudioSocket streaming listener (from `STREAMING_LISTEN_PORT` env var, default: `8080`)

**Key Design Decisions:**
- POD_IP remains required (validated at runtime, not during config load)
- Port is optional with sensible default (8080)
- Uses existing Cobra/Viper pattern for consistency
- No changes to Kubernetes deployment manifests needed (POD_IP already exists)

## Config Struct Changes

### internal/config/main.go

**Add two new fields to the Config struct:**

```go
type Config struct {
    RabbitMQAddress         string
    PrometheusEndpoint      string
    PrometheusListenAddress string
    DatabaseDSN             string
    RedisAddress            string
    RedisPassword           string
    RedisDatabase           int
    AWSAccessKey            string
    AWSSecretKey            string
    PodIP                   string // PodIP is the IP address on which the AudioSocket streaming listener binds (typically the pod's IP in Kubernetes).
    StreamingListenPort     int    // StreamingListenPort is the TCP port on which the AudioSocket streaming listener binds (default: 8080).
}
```

**Update bindConfig() to register the new flags and environment bindings:**

```go
f.String("pod_ip", "", "Pod IP address for streaming listener")
f.Int("streaming_listen_port", 8080, "Streaming listener port")

// Add to bindings map:
bindings := map[string]string{
    // ... existing bindings ...
    "pod_ip":                "POD_IP",
    "streaming_listen_port": "STREAMING_LISTEN_PORT",
}
```

**Update LoadGlobalConfig() to load the new values:**

```go
globalConfig = Config{
    // ... existing fields ...
    PodIP:               viper.GetString("pod_ip"),
    StreamingListenPort: viper.GetInt("streaming_listen_port"),
}
```

The default value for `StreamingListenPort` is set in the flag definition (`8080`), so if neither the flag nor environment variable is provided, it will use `8080`.

## Main.go Refactoring

### cmd/transcribe-manager/main.go

**Replace the direct `os.Getenv("POD_IP")` call and hardcoded port with config access:**

**Current code (lines 138-143):**
```go
listenIP := os.Getenv("POD_IP")
if listenIP == "" {
    return fmt.Errorf("could not get the listen ip address")
}
listenAddress := fmt.Sprintf("%s:%d", listenIP, 8080)
log.Debugf("Listening address... listen_address: %s", listenAddress)
```

**New code:**
```go
if config.Get().PodIP == "" {
    return fmt.Errorf("could not get the listen ip address: POD_IP not configured")
}
listenAddress := fmt.Sprintf("%s:%d", config.Get().PodIP, config.Get().StreamingListenPort)
log.Debugf("Listening address... listen_address: %s", listenAddress)
```

**Changes:**
- Remove `listenIP` local variable
- Validate `config.Get().PodIP` directly
- Use `config.Get().StreamingListenPort` instead of hardcoded `8080`
- Improved error message mentions "POD_IP not configured" for clarity
- No other changes needed in main.go (the listenAddress is still passed to streamingHandler constructor)

**Impact:**
- No `os` package import needed just for this env var anymore
- Streaming handler continues to receive the listenAddress string as before
- Runtime behavior is identical when POD_IP is set and port is not configured (uses default 8080)

## Testing Considerations

**Unit Tests:**
- No changes needed to `pkg/streaminghandler` tests - they already accept `listenAddress` as a constructor parameter
- Existing config tests (if any) would automatically cover the new fields through the standard Viper/Cobra patterns
- Mock configurations in tests can provide any value for `PodIP` and `StreamingListenPort` without requiring actual environment variables

**Integration Testing:**
- Tests that previously needed `POD_IP` environment variable can now:
  - Either set the environment variable (works as before)
  - Or explicitly set config values if testing config loading directly
- Default port value (8080) means tests don't need to set `STREAMING_LISTEN_PORT` unless testing non-default scenarios

**Backward Compatibility:**
- Existing Kubernetes deployments already set `POD_IP` in `k8s/deployment.yml:31` - no changes needed
- New deployments can optionally set `STREAMING_LISTEN_PORT` to override default 8080
- Services using environment variables continue to work identically
- Services using CLI flags gain new `--pod_ip` and `--streaming_listen_port` options

## Documentation Updates

**CLAUDE.md updates needed:**
- Update line 100: Change from "`POD_IP`: Required for AudioSocket listening address" to mention it's now in config
- Add `STREAMING_LISTEN_PORT` to the environment variables list with description: "Optional. TCP port for AudioSocket listener (default: 8080)"

**Code comments:**
- Config struct field comments included above
- No other documentation changes needed (architecture remains the same)

## Implementation Checklist

- [ ] Update `internal/config/main.go` Config struct with new fields
- [ ] Update `bindConfig()` to add new flags and environment bindings
- [ ] Update `LoadGlobalConfig()` to load new config values
- [ ] Update `cmd/transcribe-manager/main.go` to use config instead of os.Getenv
- [ ] Update `CLAUDE.md` documentation with new environment variables
- [ ] Run tests to verify backward compatibility
- [ ] Verify service starts correctly with existing POD_IP configuration
- [ ] Test with custom STREAMING_LISTEN_PORT value
