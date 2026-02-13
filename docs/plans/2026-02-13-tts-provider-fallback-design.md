# TTS Provider Fallback Design

## Problem

When an explicit TTS provider is specified (e.g., `provider="gcp"`) and it fails, the request fails entirely with no recovery. Additionally, when no provider is specified and GCP fails, the fallback to AWS passes through the original `voice_id`, which may be a GCP-specific voice that is meaningless to AWS.

## Approach

Move fallback orchestration from `AudioCreate` (audiohandler) up to `Create` (ttshandler), where caching already lives. This ensures each provider attempt gets its own correct cache key and the returned TTS result accurately reflects the provider that produced the audio.

### Fallback Rules

| Request | First attempt | Fallback |
|---------|--------------|----------|
| `provider="gcp"` | GCP with given `voice_id` | AWS with empty `voice_id` (AWS default for language) |
| `provider="aws"` | AWS with given `voice_id` | GCP with empty `voice_id` (GCP auto-selects) |
| `provider=""` | GCP with given `voice_id` | AWS with empty `voice_id` (reset, not passthrough) |

On fallback, `voice_id` is always reset to empty. Each provider uses its own default voice selection logic for the language.

### Cache Behavior

Each provider attempt computes its own cache key: `SHA1(text + lang + provider + voiceID)`. This means:

- Primary attempt cache key: `hash(text, lang, "gcp", "en-US-Wavenet-D")`
- Fallback attempt cache key: `hash(text, lang, "aws", "")`

Each is cached independently. A cached GCP result is never confused with an AWS result. Fallback results are cached for reuse on subsequent identical fallback scenarios.

**Migration note:** Existing cache entries created with `provider=""` used `hash(text, lang, "", voiceID)`. After this change, the same request resolves to `hash(text, lang, "gcp", voiceID)`. Old cache entries become orphaned (one-time cache miss). This is acceptable since the files are regenerated on next request.

### TTS Result

The returned `tts.TTS` struct reflects the **actual** provider and voiceID that produced the audio, not the original request. This ensures callers receive accurate metadata about the generated audio.

### Prometheus Metrics

- `promSpeechLanguageTotal` tracks the **actual** provider that succeeded (not the original request).
- Add a new `promSpeechFallbackTotal` counter with label `from_provider` to track how often fallbacks occur and from which provider.

## Changes

### `bin-tts-manager/pkg/ttshandler/tts.go` — `Create()`

Refactor to orchestrate fallback with per-attempt caching:

```
1. Normalize text (once, before the attempt loop)

2. Build provider attempt list:
   - provider="gcp"  → [{gcp, voiceID}, {aws, ""}]
   - provider="aws"  → [{aws, voiceID}, {gcp, ""}]
   - provider=""     → [{gcp, voiceID}, {aws, ""}]

3. For each attempt:
   a. Compute cache key: filenameHashGenerator(text, lang, attempt.provider, attempt.voiceID)
   b. Get filepath and mediaFilepath from bucket handler
   c. Check if file exists (cache hit) → return TTS result with attempt.provider/attempt.voiceID
   d. Call AudioCreate(attempt.provider, attempt.voiceID) → if success → return TTS result
   e. Log failure, if not last attempt increment promSpeechFallbackTotal, continue

4. All attempts failed → return combined error
```

A concrete attempt struct for clarity:

```go
type providerAttempt struct {
    provider tts.Provider
    voiceID  string
}
```

### `bin-tts-manager/pkg/audiohandler/audio.go` — `AudioCreate()`

Simplify to single-provider dispatch only (no fallback logic):

```go
switch provider {
case tts.ProviderGCP:  return h.gcpAudioCreate(...)
case tts.ProviderAWS:  return h.awsAudioCreate(...)
default:               return fmt.Errorf("unsupported provider: %s", provider)
}
```

The empty provider case (`""`) is removed — it is now handled by `ttshandler.Create()`.

### Test Updates

- **`ttshandler/tts_test.go`**: Add test cases for:
  - Primary succeeds (no fallback)
  - Primary fails, fallback cache hit
  - Primary fails, fallback audio creation succeeds
  - Both providers fail (combined error)
  - Empty provider resolves to GCP first
- **`audiohandler/audio_test.go`**: Update to reflect simplified `AudioCreate` — empty provider now returns error.
- Existing `aws_test.go` and `gcp_test.go` tests remain valid.

## Services Affected

Only `bin-tts-manager`. No changes to call-manager, flow-manager, or any other service. The RPC interface (`TTSV1SpeecheCreate`) remains unchanged.
