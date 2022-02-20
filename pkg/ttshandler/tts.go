package ttshandler

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	bucketDirectory = "tts"
)

// TTSCreate creates audio and upload it to the bucket.
// Returns downloadable link string
func (h *ttsHandler) TTSCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "TTSCreate",
		"call_id": callID,
	})
	log.Debugf("Creating TTS. lang: %s, gender: %s, text: %s", lang, gender, text)

	// create hash/target
	filename := h.filenameHashGenerator(text, lang, gender)
	target := fmt.Sprintf("%s/%s", bucketDirectory, filename)
	log = log.WithFields(
		logrus.Fields{
			"filename": filename,
			"target":   target,
		},
	)
	log.Debugf("Creating a new tts target. target: %s", target)

	// check exists
	if h.bucketHandler.FileExist(target) {
		log.Infof("The target file is already exsits. target: %s", target)
		return target, nil
	}

	// create audio
	err := h.audioHandler.AudioCreate(ctx, callID, text, lang, gender, filename)
	if err != nil {
		log.Errorf("Could not create audio. err: %v", err)
		return "", fmt.Errorf("could not create audio. err: %v", err)
	}
	defer os.Remove(filename)

	// upload to bucket
	if err := h.bucketHandler.FileUpload(filename, target); err != nil {
		log.Errorf("Could not upload the file to the bucket. err: %v", err)
		return "", fmt.Errorf("could not upload the file to the bucket. err: %v", err)
	}
	log.Debugf("Created and uploaded tts wav file to the bucket correctly. target: %s", target)

	return target, nil
}

// filenameHashGenerator generates hashed filename for tts wav file.
func (h *ttsHandler) filenameHashGenerator(text string, lang string, gender string) string {
	s := fmt.Sprintf("%s%s%s", text, lang, gender)

	sh1 := sha1.New()
	sh1.Write([]byte(s))
	bs := sh1.Sum(nil)

	return fmt.Sprintf("%x.wav", bs)
}
