package ttshandler

import (
	"context"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/models/tts"
)

// Create creates audio and upload it to the bucket.
// Returns downloadable link string
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

	// track language usage
	promSpeechLanguageTotal.WithLabelValues(lang, string(provider)).Inc()

	// normalize text
	normalizedText, err := h.normalizeText(ctx, text)
	if err != nil {
		log.Errorf("Could not normalize the text.")
		promSpeechRequestTotal.WithLabelValues("error").Inc()
		return nil, errors.Wrap(err, "could not normalize the text")
	}
	log.WithField("normalized_text", normalizedText).Debugf("The text has normalized.")

	// create hash/target/result
	filename := h.filenameHashGenerator(normalizedText, lang, provider, voiceID)
	osFilepath := h.bucketHandler.OSGetFilepath(ctx, filename)
	mediaFilepath := h.bucketHandler.OSGetMediaFilepath(ctx, filename)
	res := &tts.TTS{
		Provider:      provider,
		VoiceID:       voiceID,
		Text:          normalizedText,
		Language:      lang,
		MediaFilepath: mediaFilepath,
	}

	log = log.WithFields(logrus.Fields{
		"filename": filename,
		"filepath": osFilepath,
		"tts":      res,
	})
	log.Debugf("Creating a new tts target. target: %s", osFilepath)

	// check exists
	if h.bucketHandler.OSFileExist(ctx, osFilepath) {
		log.Infof("The target file is already exsits. target: %s", osFilepath)
		promSpeechRequestTotal.WithLabelValues("cache_hit").Inc()
		return res, nil
	}

	// create audio
	start := time.Now()
	if errCreate := h.audioHandler.AudioCreate(ctx, callID, normalizedText, lang, provider, voiceID, osFilepath); errCreate != nil {
		log.Errorf("Could not create audio. err: %v", errCreate)
		promSpeechRequestTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not create audio. err: %v", errCreate)
	}
	promSpeechCreateDurationSeconds.WithLabelValues().Observe(time.Since(start).Seconds())
	promSpeechRequestTotal.WithLabelValues("created").Inc()
	log.Debugf("Created tts wav file to the bucket correctly. target: %s", osFilepath)

	return res, nil
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
