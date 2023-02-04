package ttshandler

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/models/tts"
)

const (
	bucketDirectory = "tts"
)

// Create creates audio and upload it to the bucket.
// Returns downloadable link string
func (h *ttsHandler) Create(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender) (*tts.TTS, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Create",
		"call_id": callID,
	})
	log.Debugf("Creating TTS. lang: %s, gender: %s, text: %s", lang, gender, text)

	// create hash/target/result
	filename := h.filenameHashGenerator(text, lang, gender)
	filepath := fmt.Sprintf("%s/%s", bucketDirectory, filename)
	res := &tts.TTS{
		Gender:          gender,
		Text:            text,
		Language:        lang,
		MediaBucketName: h.bucketHandler.GetBucketName(),
		MediaFilepath:   filepath,
	}

	log = log.WithFields(
		logrus.Fields{
			"filename": filename,
			"filepath": filepath,
			"tts":      res,
		},
	)
	log.Debugf("Creating a new tts target. target: %s", filepath)

	// check exists
	if h.bucketHandler.FileExist(ctx, filepath) {
		log.Infof("The target file is already exsits. target: %s", filepath)
		return res, nil
	}

	// create audio
	err := h.audioHandler.AudioCreate(ctx, callID, text, lang, gender, filename)
	if err != nil {
		log.Errorf("Could not create audio. err: %v", err)
		return nil, fmt.Errorf("could not create audio. err: %v", err)
	}
	defer os.Remove(filename)

	// upload to bucket
	if err := h.bucketHandler.FileUpload(ctx, filename, filepath); err != nil {
		log.Errorf("Could not upload the file to the bucket. err: %v", err)
		return nil, fmt.Errorf("could not upload the file to the bucket. err: %v", err)
	}
	log.Debugf("Created and uploaded tts wav file to the bucket correctly. target: %s", filepath)

	return res, nil
}

// filenameHashGenerator generates hashed filename for tts wav file.
func (h *ttsHandler) filenameHashGenerator(text string, lang string, gender tts.Gender) string {
	s := fmt.Sprintf("%s%s%s", text, lang, gender)

	sh1 := sha1.New()
	sh1.Write([]byte(s))
	bs := sh1.Sum(nil)

	return fmt.Sprintf("%x.wav", bs)
}
