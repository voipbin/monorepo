# Validate tts_type and stt_type Input

Date: 2026-03-03

## Problem

The ai-manager accepts any string for `tts_type` and `stt_type` fields without validation. Invalid values like `"gcp"` (instead of `"google"`) are stored in the database and only fail at Pipecat runtime when the service tries to instantiate the provider. This causes late, confusing errors instead of immediate feedback at the API layer.

## Approach

Add `IsValid()` methods on `TTSType` and `STTType` using a map-based approach. Validate in `aihandler.Create()` and `aihandler.Update()`, consistent with the existing `engineModel` validation pattern. Return error messages that include the list of valid values.

## Design

### Map-based validation on TTSType and STTType

A package-level map serves as the single source of truth for valid values. Both `IsValid()` and `ValidValues()` derive from it.

```go
var validTTSTypes = map[TTSType]bool{
    TTSTypeNone: true, TTSTypeAsync: true, TTSTypeAWS: true,
    TTSTypeAzure: true, TTSTypeCartesia: true, TTSTypeDeepgram: true,
    TTSTypeElevenLabs: true, TTSTypeFish: true, TTSTypeGoogle: true,
    TTSTypeGroq: true, TTSTypeHume: true, TTSTypeInworld: true,
    TTSTypeLMNT: true, TTSTypeMiniMax: true, TTSTypeNeuphonic: true,
    TTSTypeNvidiaRiva: true, TTSTypeOpenAI: true, TTSTypePiper: true,
    TTSTypePlayHT: true, TTSTypeRime: true, TTSTypeSarvam: true,
    TTSTypeXTTS: true,
}

func (t TTSType) IsValid() bool { return validTTSTypes[t] }

func (t TTSType) ValidValues() []string {
    res := make([]string, 0, len(validTTSTypes))
    for k := range validTTSTypes {
        if k != TTSTypeNone {
            res = append(res, string(k))
        }
    }
    sort.Strings(res)
    return res
}
```

Same pattern for STTType with its own map and methods.

### Validation in aihandler

Add validation after the existing `engineModel` check in both `Create()` and `Update()`:

```go
if !ttsType.IsValid() {
    return nil, fmt.Errorf("invalid tts_type: %s. valid values: %s", ttsType, strings.Join(ttsType.ValidValues(), ", "))
}
if !sttType.IsValid() {
    return nil, fmt.Errorf("invalid stt_type: %s. valid values: %s", sttType, strings.Join(sttType.ValidValues(), ", "))
}
```

### Validation in ai-control CLI

Add the same validation in `cmd/ai-control/main.go` for both `runCreate` and `runUpdate`, alongside the existing `engineModel` check.

### Maintenance

When adding a new TTS/STT provider, update 2 places:
1. Add the `const` value
2. Add it to the corresponding `validTTSTypes`/`validSTTTypes` map

### What does NOT change

- No OpenAPI changes (enum constraints already correct)
- No database changes
- No changes to other services
- No migration for existing invalid data (only prevents new invalid values)

## Files Changed

| File | Change |
|------|--------|
| `bin-ai-manager/models/ai/main.go` | Add `validTTSTypes` map, `validSTTTypes` map, `IsValid()` and `ValidValues()` methods on both types |
| `bin-ai-manager/models/ai/main_test.go` | Add table-driven tests for `IsValid()` and `ValidValues()` |
| `bin-ai-manager/pkg/aihandler/chatbot.go` | Add validation calls in `Create()` and `Update()` |
| `bin-ai-manager/pkg/aihandler/chatbot_test.go` | Add test cases for invalid tts_type/stt_type |
| `bin-ai-manager/cmd/ai-control/main.go` | Add validation for CLI tool |
