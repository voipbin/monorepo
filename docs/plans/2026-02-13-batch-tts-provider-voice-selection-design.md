# Batch TTS Provider and Voice Selection

## Problem

The batch TTS system currently has no caller control over which TTS provider (GCP or AWS) is used, and voice selection is limited to a hard-coded mapping of language + gender. Callers cannot specify a particular voice ID, and the Gender field adds unnecessary complexity now that voice IDs are the preferred selection mechanism.

## Approach

Add `Provider` and `VoiceID` fields to the TTS request model, remove the `Gender` field, and simplify the default voice maps to one voice per language.

### Fallback behavior
1. **Provider + VoiceID specified** - use exactly that provider and voice, no fallback
2. **Provider specified, no VoiceID** - use that provider with the default voice map (one voice per language)
3. **Neither specified** - current behavior (try GCP first, fall back to AWS)

### Cache key
Change from `SHA1(text + language + gender)` to `SHA1(text + language + provider + voice_id)` to ensure different provider/voice combos produce separate cache entries.

## Files to Change

### bin-tts-manager

**models/tts/tts.go**
- Add `Provider` type and constants (`ProviderGCP`, `ProviderAWS`)
- Add `Provider` and `VoiceID` fields to `TTS` struct
- Remove `Gender` type and constants

**pkg/listenhandler/models/request/speeches.go**
- Replace `Gender` with `Provider` and `VoiceID` in `V1DataSpeechesPost`

**pkg/listenhandler/v1_speeches.go**
- Pass `Provider` and `VoiceID` (instead of `Gender`) to `ttsHandler.Create`

**pkg/listenhandler/v1_speeches_test.go**
- Update test data to use `Provider`/`VoiceID` instead of `Gender`

**pkg/ttshandler/main.go**
- Update `TTSHandler.Create` signature: remove `gender`, add `provider` and `voiceID`
- Update `promSpeechLanguageTotal` metric: remove `gender` label, add `provider` label

**pkg/ttshandler/tts.go**
- Update `Create` implementation: use new parameters
- Update `filenameHashGenerator`: hash `text + lang + provider + voiceID`

**pkg/audiohandler/main.go**
- Update `AudioHandler.AudioCreate` signature: remove `gender`, add `provider` and `voiceID`

**pkg/audiohandler/audio.go**
- Add provider routing: if provider specified, call only that provider; otherwise try GCP then AWS
- Pass `voiceID` to provider methods

**pkg/audiohandler/gcp.go**
- Change `gcpGetVoiceName` to one default voice per language (no gender dimension)
- Accept `voiceID` parameter; if non-empty, use it directly instead of the default map
- Remove `ssmlGender` selection logic

**pkg/audiohandler/aws.go**
- Change `awsGetVoiceID` to one default voice per language (no gender dimension)
- Accept `voiceID` parameter; if non-empty, use it directly as `types.VoiceId`

### bin-common-handler

**pkg/requesthandler/main.go**
- Update `TTSV1SpeecheCreate` interface signature: remove `gender`, add `provider` and `voiceID`

**pkg/requesthandler/tts_speeches.go**
- Update implementation to send `Provider` and `VoiceID` in the request

**pkg/requesthandler/tts_speeches_test.go**
- Update test data

### bin-call-manager (caller)

**pkg/callhandler/main.go**
- Update `CallHandler.Talk` interface signature: remove `gender`, add `provider` and `voiceID`

**pkg/callhandler/media.go**
- Update `Talk` method: remove `gender` parameter, add `provider` and `voiceID`
- Update `TTSV1SpeecheCreate` call

**pkg/callhandler/media_test.go**
- Update test data

**pkg/callhandler/action.go**
- Update `actionExecuteTalk` to pass `option.Provider` and `option.VoiceID` instead of `option.Gender`

**pkg/callhandler/action_test.go**
- Update test data for TTS-related action tests

**pkg/listenhandler/v1_calls.go**
- Update `processV1CallsIDTalkPost` to pass `req.Provider` and `req.VoiceID` instead of `req.Gender`

**pkg/listenhandler/models/request/calls.go**
- Update `V1DataCallsIDTalkPost`: remove `Gender`, add `Provider` and `VoiceID`

### bin-flow-manager (action option)

**models/action/option.go**
- Update `OptionTalk`: add `Provider` and `VoiceID` fields, keep `Gender` for backward compatibility (existing flows may have it configured, just won't be forwarded to TTS)

**models/action/option_test.go**
- Update test data for OptionTalk

### Mock regeneration
- `bin-tts-manager/pkg/ttshandler/mock_main.go`
- `bin-tts-manager/pkg/audiohandler/mock_main.go`
- `bin-common-handler/pkg/requesthandler/mock_main.go`
- `bin-call-manager/pkg/callhandler/mock_main.go`

## Default Voice Maps (post-change)

### GCP (one Wavenet voice per language)
| Language | Voice |
|----------|-------|
| en-US | en-US-Wavenet-F |
| en-GB | en-GB-Wavenet-A |
| de-DE | de-DE-Wavenet-F |
| fr-FR | fr-FR-Wavenet-E |
| es-ES | es-ES-Wavenet-E |
| it-IT | it-IT-Wavenet-E |
| ja-JP | ja-JP-Wavenet-C |
| ko-KR | ko-KR-Wavenet-C |

### AWS (one Polly voice per language)
| Language | Voice |
|----------|-------|
| en-US | Joanna |
| en-GB | Amy |
| de-DE | Marlene |
| fr-FR | Celine |
| es-ES | Conchita |
| it-IT | Carla |
| ja-JP | Mizuki |
| ko-KR | Seoyeon |
| pt-BR | Camila |
| ru-RU | Tatyana |
| zh-CN | Zhiyu |

## Verification

Run full verification for:
- `bin-tts-manager`
- `bin-common-handler`
- `bin-call-manager`
- `bin-flow-manager`

All four are affected by the signature or model changes.
