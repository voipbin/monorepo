# Configurable STT Provider Priority Design

**Date:** 2026-01-06
**Status:** Approved

## Overview and Goals

This design adds a configurable priority system for STT (Speech-to-Text) providers in transcribe-manager, replacing the current hardcoded "GCP first, then AWS" order with an environment variable-driven approach.

**Current Behavior:**
- Hardcoded order in `pkg/streaminghandler/run.go:72-77`
- Always tries GCP first, then AWS as fallback
- No way to prefer AWS or change the order

**New Behavior:**
- Environment variable `STT_PROVIDER_PRIORITY` defines the order (e.g., `"GCP,AWS"` or `"AWS,GCP"`)
- Tries providers in specified order with fallback
- Default value `"GCP,AWS"` maintains backward compatibility
- Strict validation: fails startup if any listed provider is unknown or not initialized

**Key Design Decisions:**
1. **Comma-separated format**: Simple to parse, follows common conventions (like PATH)
2. **Fallback on failure**: If first provider fails during streaming, tries next provider automatically
3. **Backward compatible**: Default value preserves current behavior for existing deployments
4. **Fail-fast validation**: Catches configuration errors at startup, not during production streaming

**Configuration Examples:**
- `STT_PROVIDER_PRIORITY="AWS,GCP"` - Prefer AWS, fall back to GCP
- `STT_PROVIDER_PRIORITY="GCP,AWS"` - Current behavior (also the default)
- Not set → defaults to `"GCP,AWS"`

## Configuration and Validation

**Add to internal/config package:**

New field in Config struct:
```go
STTProviderPriority string // STTProviderPriority is the comma-separated list of STT providers in priority order (e.g., "GCP,AWS"). Default: "GCP,AWS"
```

**Flag and environment binding:**
- Flag: `--stt_provider_priority` with default `"GCP,AWS"`
- Environment variable: `STT_PROVIDER_PRIORITY`
- Default value: `"GCP,AWS"` (maintains current behavior)

**Validation logic in NewStreamingHandler:**

1. **Parse the priority string**: Split by comma, trim whitespace
   - `"GCP,AWS"` → `["GCP", "AWS"]`
   - `" AWS , GCP "` → `["AWS", "GCP"]` (handles extra spaces)

2. **Validate each provider name**: Must be exactly "GCP" or "AWS" (case-sensitive)
   - Unknown provider → **fail startup** with error: `"Unknown STT provider in priority list: AZURE"`
   - Empty priority list → **fail startup** with error: `"STT_PROVIDER_PRIORITY cannot be empty"`

3. **Check provider initialization**: Each listed provider must be successfully initialized
   - Priority includes "GCP" but `gcpClient == nil` → **fail startup** with error: `"STT provider 'GCP' listed in priority but not initialized (check credentials)"`
   - Priority includes "AWS" but `awsClient == nil` → **fail startup** with error: `"STT provider 'AWS' listed in priority but not initialized (check credentials)"`

4. **Verify at least one provider remains**: After validation, at least one provider must be available
   - If none → **fail startup** (already handled by existing validation)

**Error message examples:**
- `"Invalid STT_PROVIDER_PRIORITY: unknown provider 'AZURE'. Valid providers: GCP, AWS"`
- `"Invalid STT_PROVIDER_PRIORITY: provider 'AWS' is listed but AWS credentials are not configured"`
- `"Invalid STT_PROVIDER_PRIORITY: provider 'GCP' is listed but GCP credentials are not available"`

This strict validation ensures configuration errors are caught immediately at service startup rather than failing during production streaming.

## Implementation Changes

### Type-Safe Constants

**File: pkg/streaminghandler/main.go**

Add type and constants at the top of the file:

```go
// STTProvider represents a speech-to-text provider type
type STTProvider string

const (
	STTProviderGCP STTProvider = "GCP"
	STTProviderAWS STTProvider = "AWS"
)
```

### Validation in NewStreamingHandler

Update validation in `NewStreamingHandler()` after client initialization:

```go
// Parse and validate STT provider priority
priorityList := strings.Split(config.Get().STTProviderPriority, ",")
var validatedProviders []STTProvider

for _, providerStr := range priorityList {
	providerStr = strings.TrimSpace(providerStr)
	provider := STTProvider(providerStr)

	// Validate provider name
	if provider != STTProviderGCP && provider != STTProviderAWS {
		log.Errorf("Unknown STT provider in priority list: %s. Valid providers: %s, %s",
			providerStr, STTProviderGCP, STTProviderAWS)
		return nil
	}

	// Validate provider is initialized
	if provider == STTProviderGCP && gcpClient == nil {
		log.Errorf("STT provider '%s' listed in priority but not initialized (check GCP credentials)", STTProviderGCP)
		return nil
	}
	if provider == STTProviderAWS && awsClient == nil {
		log.Errorf("STT provider '%s' listed in priority but not initialized (check AWS credentials)", STTProviderAWS)
		return nil
	}

	validatedProviders = append(validatedProviders, provider)
}

if len(validatedProviders) == 0 {
	log.Error("No valid STT providers in priority list")
	return nil
}

log.Infof("STT provider priority: %s", strings.Join(validatedProviders, " → "))
```

Update the handler struct:
```go
type streamingHandler struct {
	// ... existing fields ...
	providerPriority []STTProvider // Validated list of providers in priority order
}
```

### Handler Building

**File: pkg/streaminghandler/run.go**

Replace the hardcoded handlers building (lines 71-77) with priority-based logic:

```go
// Build handlers list based on configured priority
handlers := []func(*streaming.Streaming, net.Conn) error{}
for _, provider := range h.providerPriority {
	switch provider {
	case STTProviderGCP:
		handlers = append(handlers, h.gcpRun)
	case STTProviderAWS:
		handlers = append(handlers, h.awsRun)
	}
}
```

**Behavior:**
- If priority is `["GCP", "AWS"]`: tries `h.gcpRun`, then `h.awsRun` on failure
- If priority is `["AWS", "GCP"]`: tries `h.awsRun`, then `h.gcpRun` on failure
- The existing fallback loop (lines 79-95) remains unchanged - it already tries each handler in order

## Documentation Updates

### CLAUDE.md Changes

**Add to environment variables section (around line 100):**
```markdown
- `STT_PROVIDER_PRIORITY`: Optional. Comma-separated list of STT providers in priority order (default: "GCP,AWS"). Valid values: GCP, AWS. Examples: "GCP,AWS", "AWS,GCP"
```

**Update STT Provider Requirements section (around line 105):**
```markdown
**STT Provider Requirements:**
- At least one STT provider must be configured (GCP or AWS)
- GCP: Uses Application Default Credentials (ADC) - can be from service account key, gcloud CLI, GKE metadata server, etc.
- AWS: Requires both `AWS_ACCESS_KEY` and `AWS_SECRET_KEY` environment variables
- Provider priority can be configured via `STT_PROVIDER_PRIORITY` (default: "GCP,AWS")
- All providers listed in `STT_PROVIDER_PRIORITY` must be properly configured, or service will fail to start
- Service fails to start if no providers are available or if priority configuration is invalid
```

## Testing Considerations

### Unit Tests for Validation Logic

Test cases to add:
- Valid priority strings: `"GCP,AWS"`, `"AWS,GCP"`, `" GCP , AWS "` (with spaces)
- Invalid provider names: `"AZURE"`, `"GCP,INVALID"`, `""`
- Provider not initialized: Priority includes "AWS" but awsClient is nil
- Empty priority list after validation

### Integration Tests

Test scenarios:
- Default behavior (not set) → uses "GCP,AWS"
- Priority="AWS,GCP" → tries AWS first
- Priority="GCP" with only GCP initialized → works
- Priority="GCP,AWS" but AWS not initialized → fails startup

### Existing Tests

Updates needed:
- Update `NewStreamingHandler` tests to include `STTProviderPriority` in config mock
- Verify existing streaming tests still pass with default priority

## Backward Compatibility

**100% Backward Compatible:**
- Default value `"GCP,AWS"` maintains current behavior
- Existing deployments without `STT_PROVIDER_PRIORITY` work unchanged
- No changes to Kubernetes deployment.yml needed (optional override)
- No API changes
- No database schema changes

## Implementation Checklist

- [ ] Add `STTProvider` type and constants to `pkg/streaminghandler/main.go`
- [ ] Add `STTProviderPriority` field to `internal/config.Config`
- [ ] Add flag and environment variable binding in `internal/config`
- [ ] Add validation logic in `NewStreamingHandler()`
- [ ] Add `providerPriority []STTProvider` field to `streamingHandler` struct
- [ ] Update handler building in `pkg/streaminghandler/run.go`
- [ ] Update CLAUDE.md documentation
- [ ] Add unit tests for validation logic
- [ ] Add integration tests for priority ordering
- [ ] Update existing tests with new config field
- [ ] Manual verification with different priority configurations
