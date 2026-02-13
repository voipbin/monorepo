package ttshandler

import (
	"context"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/models/tts"
)

// providerAttempt defines a single provider attempt with its voice configuration.
type providerAttempt struct {
	provider tts.Provider
	voiceID  string
}

// buildAttempts returns the ordered list of provider attempts for fallback.
// On fallback, voice_id is reset to empty so the alternative provider uses its own default.
func buildAttempts(provider tts.Provider, voiceID string) []providerAttempt {
	switch provider {
	case tts.ProviderGCP:
		return []providerAttempt{
			{provider: tts.ProviderGCP, voiceID: voiceID},
			{provider: tts.ProviderAWS, voiceID: ""},
		}
	case tts.ProviderAWS:
		return []providerAttempt{
			{provider: tts.ProviderAWS, voiceID: voiceID},
			{provider: tts.ProviderGCP, voiceID: ""},
		}
	default:
		// empty or unknown provider: default to GCP first, fallback to AWS
		return []providerAttempt{
			{provider: tts.ProviderGCP, voiceID: voiceID},
			{provider: tts.ProviderAWS, voiceID: ""},
		}
	}
}

// Create creates audio and uploads it to the bucket.
// It tries the primary provider first; on failure, falls back to an alternative provider
// with an empty voice_id (provider default). Each attempt has its own cache key.
func (h *ttsHandler) Create(ctx context.Context, callID uuid.UUID, text string, lang string, provider tts.Provider, voiceID string) (*tts.TTS, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Create",
		"call_id":  callID,
		"text":     text,
		"language": lang,
		"provider": provider,
		"voice_id": voiceID,
	})
	log.Debugf("Creating TTS. lang: %s, provider: %s, voice_id: %s, text: %s", lang, provider, voiceID, text)

	// normalize text once before the attempt loop
	normalizedText, err := h.normalizeText(ctx, text)
	if err != nil {
		log.Errorf("Could not normalize the text.")
		promSpeechRequestTotal.WithLabelValues("error").Inc()
		return nil, errors.Wrap(err, "could not normalize the text")
	}
	log.WithField("normalized_text", normalizedText).Debugf("The text has normalized.")

	attempts := buildAttempts(provider, voiceID)
	var errs []string

	for i, attempt := range attempts {
		attemptLog := log.WithFields(logrus.Fields{
			"attempt_provider": attempt.provider,
			"attempt_voice_id": attempt.voiceID,
			"attempt_index":    i,
		})

		// compute per-attempt cache key
		filename := h.filenameHashGenerator(normalizedText, lang, attempt.provider, attempt.voiceID)
		osFilepath := h.bucketHandler.OSGetFilepath(ctx, filename)
		mediaFilepath := h.bucketHandler.OSGetMediaFilepath(ctx, filename)

		res := &tts.TTS{
			Provider:      attempt.provider,
			VoiceID:       attempt.voiceID,
			Text:          normalizedText,
			Language:      lang,
			MediaFilepath: mediaFilepath,
		}

		// cache hit — return immediately
		if h.bucketHandler.OSFileExist(ctx, osFilepath) {
			attemptLog.Infof("Cache hit for provider %s. target: %s", attempt.provider, osFilepath)
			promSpeechRequestTotal.WithLabelValues("cache_hit").Inc()
			promSpeechLanguageTotal.WithLabelValues(lang, string(attempt.provider)).Inc()
			return res, nil
		}

		// create audio
		start := time.Now()
		if errCreate := h.audioHandler.AudioCreate(ctx, callID, normalizedText, lang, attempt.provider, attempt.voiceID, osFilepath); errCreate != nil {
			attemptLog.Warnf("Provider %s failed. err: %v", attempt.provider, errCreate)
			errs = append(errs, fmt.Sprintf("%s: %v", attempt.provider, errCreate))

			// track fallback if this is not the last attempt
			if i < len(attempts)-1 {
				promSpeechFallbackTotal.WithLabelValues(string(attempt.provider)).Inc()
			}
			continue
		}

		promSpeechCreateDurationSeconds.WithLabelValues().Observe(time.Since(start).Seconds())
		promSpeechRequestTotal.WithLabelValues("created").Inc()
		promSpeechLanguageTotal.WithLabelValues(lang, string(attempt.provider)).Inc()
		attemptLog.Debugf("Created tts wav file to the bucket correctly. target: %s", osFilepath)

		return res, nil
	}

	// all attempts failed
	promSpeechRequestTotal.WithLabelValues("error").Inc()
	return nil, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
}

// filenameHashGenerator generates hashed filename for tts wav file.
func (h *ttsHandler) filenameHashGenerator(text string, lang string, provider tts.Provider, voiceID string) string {
	log := logrus.WithFields(logrus.Fields{
		"func": "filenameHashGenerator",
	})

	s := fmt.Sprintf("%s%s%s%s", text, lang, provider, voiceID)
	start := time.Now()

	sh1 := sha1.New()
	sh1.Write([]byte(s))
	bs := sh1.Sum(nil)

	res := fmt.Sprintf("%x.wav", bs)
	elapsed := time.Since(start)

	log.Debugf("Hashing duration. res: %s, duration: %s", res, elapsed)
	promHashProcessTime.WithLabelValues().Observe(float64(elapsed.Milliseconds()))

	return res
}

// normalizeText returns normalized ssml
func (h *ttsHandler) normalizeText(ctx context.Context, ssml string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "normalizeSSML",
		"ssml": ssml,
	})

	res := ssml
	if valid := h.isValidSSML(res); !valid {
		log.Debugf("The text is not valid ssml. adding the default ")
		res = fmt.Sprintf("<speak>%s</speak>", res)

		// validate again
		if reValid := h.isValidSSML(res); !reValid {
			log.Errorf("Could not pass the ssml validation.")
			return "", fmt.Errorf("could not pass the ssml validation")
		}
	}

	return res, nil
}

// isValidSSML returns true if the given text is valid ssml
func (h *ttsHandler) isValidSSML(text string) bool {

	return xml.Unmarshal([]byte(text), new(any)) == nil
}
