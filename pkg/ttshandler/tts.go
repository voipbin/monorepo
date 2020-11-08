package ttshandler

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	bucketDirectory = "tts"
)

// TTSCreate creates audio and upload it to the bucket.
func (h *ttsHandler) TTSCreate(text string, lang string, gender string) (string, error) {

	// create hash/target
	filename := h.filenameHashGenerator(text, lang, gender)
	target := fmt.Sprintf("%s/%s", bucketDirectory, filename)

	// check exists
	if h.bucketHandler.FileExist(target) == true {
		logrus.Infof("The file is already exsits.")
		return target, nil
	}

	// create audio
	err := h.audioHandler.AudioCreate(text, lang, gender, filename)
	if err != nil {
		logrus.Errorf("Could not create audio. err: %v", err)
		return "", fmt.Errorf("could not create audio. err: %v", err)
	}
	defer os.Remove(filename)

	// upload to bucket
	if err := h.bucketHandler.FileUpload(filename, target); err != nil {
		logrus.Errorf("Could not upload the file to the bucket. err: %v", err)
		return "", fmt.Errorf("could not upload the file to the bucket. err: %v", err)
	}
	logrus.Debugf("Created and uploaded tts wav file to the bucket correctly. target: %s", target)

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
