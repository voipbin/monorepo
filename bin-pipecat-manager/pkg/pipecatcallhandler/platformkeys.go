package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_platformkeys.go -source platformkeys.go -build_flags=-mod=mod

import (
	"context"
	"sort"
	"strings"

	"github.com/gofrs/uuid"

	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-pipecat-manager/internal/configkeys"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

// PlatformKeyResolver resolves the platform keys that a session uses. It takes the names of the
// configuration keys (the constants of configkeys) and returns their values by name.
//
// It does not return an error: a key that can not be read is left out of the result (the
// implementation logs a warning without the value), and the runner then falls back to its own
// environment for that key. A key without a value is left out too.
type PlatformKeyResolver interface {
	Resolve(ctx context.Context, pipecatcallID uuid.UUID, keyNames []string) map[string]amai.RedactedString
}

// llmServiceName returns the service part of a runner llm type, the way the runner reads it: the
// type is lower cased and split at the first "." (or at the first ":" when there is no ".").
// A type without a separator has no service name (the runner rejects it).
func llmServiceName(runnerType string) string {
	t := strings.ToLower(runnerType)
	if name, _, ok := strings.Cut(t, "."); ok {
		return name
	}
	if name, _, ok := strings.Cut(t, ":"); ok {
		return name
	}

	return ""
}

// llmKeyName returns the configuration key that the runner reads for an llm type, or "" when the
// type uses no stored key. The customer key of a session is not part of this: OpenAI, Grok and
// Gemini prefer the customer key in the runner, the other services never use it.
//
// platform_openrouter and local read the stored key only. A customer's own OpenRouter model
// (service "openrouter") and a customer specified endpoint use the key of the customer only, so
// they need no platform key.
func llmKeyName(runnerType string) string {
	switch llmServiceName(runnerType) {
	case "openai":
		return configkeys.KeyOpenAIAPIKey
	case "grok":
		return configkeys.KeyXAIAPIKey
	case "gemini":
		return configkeys.KeyGoogleAPIKey
	case "platform_openrouter":
		return configkeys.KeyOpenRouterAPIKey
	case "local":
		return configkeys.KeyLocalLLMEndpoints
	default:
		return ""
	}
}

// sttKeyName returns the configuration key of a stt service, or "" (Google STT uses a service
// account).
func sttKeyName(sttType string) string {
	if strings.ToLower(sttType) == "deepgram" {
		return configkeys.KeyDeepgramAPIKey
	}

	return ""
}

// ttsKeyName returns the configuration key of a tts service, or "" (Google TTS uses a service
// account).
func ttsKeyName(ttsType string) string {
	switch strings.ToLower(ttsType) {
	case "cartesia":
		return configkeys.KeyCartesiaAPIKey
	case "elevenlabs":
		return configkeys.KeyElevenLabsAPIKey
	default:
		return ""
	}
}

// platformKeyNames returns the sorted, de-duplicated names of the stored keys that the given
// vendors need. A vendor that the session does not use is never in the result.
func platformKeyNames(llmTypes, sttTypes, ttsTypes []string) []string {
	set := map[string]struct{}{}
	add := func(name string) {
		if name != "" {
			set[name] = struct{}{}
		}
	}
	for _, t := range llmTypes {
		add(llmKeyName(t))
	}
	for _, t := range sttTypes {
		add(sttKeyName(t))
	}
	for _, t := range ttsTypes {
		add(ttsKeyName(t))
	}

	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// sessionPlatformKeys resolves the platform keys of a session for the runner request. A single AI
// session and a team session go through this one function, so the same configuration gets the same
// map.
//
// A team session is read from its members: the top level llm type of the session is not read by
// the runner. A member that was rejected (an empty llm type) is skipped. The members' stt and tts
// are used only when the call itself has them (text mode sessions build neither), which is the
// rule of the runner.
//
// It returns nil when no resolver is set, or when the session uses no stored key.
func (h *pipecatcallHandler) sessionPlatformKeys(ctx context.Context, pc *pipecatcall.Pipecatcall, se *pipecatcall.Session, team *resolvedTeamData) map[string]amai.RedactedString {
	if h.platformKeys == nil {
		return nil
	}

	var llmTypes, sttTypes, ttsTypes []string
	if team != nil {
		for _, m := range team.Members {
			if m.AI.LLMType != "" {
				llmTypes = append(llmTypes, m.AI.LLMType)
			}
			if pc.STTType != "" {
				sttTypes = append(sttTypes, m.AI.STTType)
			}
			if pc.TTSType != "" {
				ttsTypes = append(ttsTypes, m.AI.TTSType)
			}
		}
	} else {
		llmTypes = []string{se.LLMRunnerType}
		sttTypes = []string{string(pc.STTType)}
		ttsTypes = []string{string(pc.TTSType)}
	}

	names := platformKeyNames(llmTypes, sttTypes, ttsTypes)
	if len(names) == 0 {
		return nil
	}

	return h.platformKeys.Resolve(ctx, pc.ID, names)
}
