# Optional STT Provider Configuration Design

**Date**: 2026-01-05
**Status**: Approved

## Overview

Change the transcribe-manager service to make GCP and AWS STT provider credentials optional instead of required. At least one provider must be configured, but not both. If only one provider is available, use it exclusively. If both are available, maintain the current fallback behavior (try GCP first, then AWS).

## Current Behavior

- Both GCP and AWS clients are initialized at startup in `NewStreamingHandler()`
- If either client initialization fails, the entire service fails to start
- Runtime uses a fallback loop: try GCP first, then AWS if GCP fails
- AWS credentials: Explicit via `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` environment variables
- GCP credentials: Implicit via Application Default Credentials (ADC) - can come from service account key, gcloud CLI, GKE metadata server, etc.

## Design

### 1. Startup Validation & Conditional Initialization

**Configuration Validation** (`cmd/transcribe-manager/main.go`):
- Check if at least one provider has credentials configured
- **AWS**: Validate that both `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` are non-empty
- **GCP**: Cannot pre-validate due to ADC - must attempt client creation to determine availability

**Conditional Client Initialization** (`pkg/streaminghandler/main.go:NewStreamingHandler`):
- **GCP**: Always attempt `speech.NewClient()` - if it fails, assume no credentials available
  - On failure: Set `gcpClient = nil`, log warning
  - On success: Set `gcpClient` to valid client
- **AWS**: Only attempt `awsNewClient()` if both access key and secret key are non-empty
  - If credentials empty: Set `awsClient = nil`
  - If credentials present but client creation fails: Set `awsClient = nil`, log warning
- **Final validation**: Return error only if both clients are nil (no providers available)

**Implementation**:
```go
// In NewStreamingHandler():
var gcpClient *speech.Client
var awsClient *transcribestreaming.Client

// Always try GCP (ADC-based)
gcpClient, err := speech.NewClient(context.Background())
if err != nil {
    log.Warnf("GCP client initialization failed (credentials not available): %v", err)
    gcpClient = nil
}

// Only try AWS if credentials are provided
if awsAccessKey != "" && awsSecretKey != "" {
    awsClient, err = awsNewClient(awsAccessKey, awsSecretKey)
    if err != nil {
        log.Warnf("AWS client initialization failed: %v", err)
        awsClient = nil
    }
} else {
    awsClient = nil
}

// Validate at least one provider is available
var providers []string
if gcpClient != nil {
    providers = append(providers, "GCP")
}
if awsClient != nil {
    providers = append(providers, "AWS")
}
if len(providers) == 0 {
    log.Error("No STT providers available - at least one provider must be configured")
    return nil
}
log.Infof("STT providers initialized: %s", strings.Join(providers, ", "))

return &streamingHandler{
    // ... other fields ...
    gcpClient: gcpClient,
    awsClient: awsClient,
    // ... other fields ...
}
```

### 2. Runtime Handler Selection & Fallback

**Dynamic Handler List** (`pkg/streaminghandler/run.go:runStart`):
- Build handlers list dynamically based on available clients
- Only include `h.gcpRun` if `h.gcpClient != nil`
- Only include `h.awsRun` if `h.awsClient != nil`
- Maintain existing fallback loop logic (tries each handler until one succeeds)

**Implementation**:
```go
// In runStart(), replace static handler list:
handlers := []func(*streaming.Streaming, net.Conn) error{}
if h.gcpClient != nil {
    handlers = append(handlers, h.gcpRun)
}
if h.awsClient != nil {
    handlers = append(handlers, h.awsRun)
}

if len(handlers) == 0 {
    log.Error("No STT providers available for transcription")
    return
}

// Existing loop continues to work
for _, handler := range handlers {
    if errRun := handler(st, conn); errRun != nil {
        log.Errorf("Handler execution failed: %v", errRun)
        continue
    }
    return
}

log.Warn("No handler executed successfully")
```

**Provider Method Updates**:
- Add nil-check at start of `gcpRun()` and `awsRun()` for defense-in-depth
- Return clear error if client is nil

```go
// In gcpRun():
if h.gcpClient == nil {
    return fmt.Errorf("GCP provider not initialized")
}

// In awsRun():
if h.awsClient == nil {
    return fmt.Errorf("AWS provider not initialized")
}
```

**Behavior**:
- If both providers available: Try GCP first, fall back to AWS on failure
- If only one provider available: Use that provider exclusively
- If no providers available: Service fails to start

### 3. Error Handling & Logging

**Startup Logging**:
- Log which providers were successfully initialized at INFO level:
  - "STT providers initialized: GCP, AWS"
  - "STT providers initialized: GCP"
  - "STT providers initialized: AWS"
- If both fail: Log ERROR and return nil from `NewStreamingHandler()`

**Runtime Logging**:
- Existing error logging in fallback loop is sufficient
- Add ERROR log if handlers list is empty (shouldn't happen if startup validation works correctly)
- Provider-specific errors already logged by `gcpRun()` and `awsRun()`

**Error Messages**:
- Startup failure: "No STT providers available - at least one provider must be configured"
- Runtime nil client: "GCP/AWS provider not initialized"
- Empty handler list: "No STT providers available for transcription"

### 4. Configuration & Testing

**Configuration**:
- No new configuration fields needed
- Existing environment variables remain:
  - `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` (optional if GCP available)
  - GCP continues using ADC (optional if AWS available)
- At least one provider must be configured

**Optional Validation Helper**:
```go
// In cmd/transcribe-manager/main.go (optional pre-flight check)
func validateProviders() {
    hasAWS := config.Get().AWSAccessKey != "" && config.Get().AWSSecretKey != ""

    if !hasAWS {
        log.Warn("AWS credentials not configured - AWS provider will be unavailable")
    }
    log.Info("GCP credentials will be validated via ADC during client initialization")

    // Final validation happens in NewStreamingHandler
}
```

**Testing Updates Required**:
- `pkg/streaminghandler/main_test.go`: Test scenarios for:
  - Both providers available
  - Only GCP available (empty AWS keys)
  - Only AWS available (GCP client creation fails)
  - Neither available (should return nil)
- `pkg/streaminghandler/start_test.go`: Update test setup to handle nil clients
- Integration tests: Verify fallback behavior with single provider

**Backwards Compatibility**:
- Existing deployments with both providers configured work unchanged
- No breaking changes to API, message format, or behavior
- Services with both credentials continue to use GCP-first fallback

## Implementation Files to Modify

1. `pkg/streaminghandler/main.go` - Conditional client initialization in `NewStreamingHandler()`
2. `pkg/streaminghandler/run.go` - Dynamic handler list in `runStart()`
3. `pkg/streaminghandler/gcp.go` - Add nil-check in `gcpRun()`
4. `pkg/streaminghandler/aws.go` - Add nil-check in `awsRun()`
5. `cmd/transcribe-manager/main.go` - Optional pre-flight validation (warning logs)
6. Tests: Update existing tests to cover new scenarios

## Success Criteria

- Service starts successfully with only GCP credentials configured
- Service starts successfully with only AWS credentials configured
- Service starts successfully with both credentials configured (existing behavior)
- Service fails to start with clear error message if neither provider is configured
- Runtime transcription works with single provider (no fallback available)
- Runtime transcription maintains fallback behavior with both providers
- Existing deployments continue to work without configuration changes
