package audiohandler

import (
	"context"
	"fmt"
	"os"

	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/models/tts"
)

// AudioCreate Creates tts audio
func (h *audioHandler) AudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AudioCreate",
		"call_id": callID,
	})
	log.WithField("text", text).Debugf("Creating a new audio. lang: %s, gender: %s, filepath: %s", lang, gender, filepath)

	voiceName := h.getVoiceName(lang, gender)
	ssmlGender := texttospeechpb.SsmlVoiceGender_NEUTRAL
	switch gender {
	case tts.GenderMale:
		ssmlGender = texttospeechpb.SsmlVoiceGender_MALE

	case tts.GenderFemale:
		ssmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	}

	// perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	req := texttospeechpb.SynthesizeSpeechRequest{

		// set the ssml input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Ssml{
				Ssml: text,
			},
		},

		// build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral")
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: lang,
			Name:         voiceName,
			SsmlGender:   ssmlGender,
		},

		// select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   defaultAudioEncoding,
			SampleRateHertz: defaultSampleRate,
		},
	}
	log.Debugf("Send speech request. language_code: %s, gender: %d, name: %s", req.Voice.LanguageCode, req.Voice.SsmlGender, voiceName)

	// send request
	resp, err := h.client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		log.Errorf("Could not get a correct response. text: %s, lang: %s, ssmlGender: %v, err: %v", text, lang, ssmlGender, err)
		return err
	}

	// create audio
	if errWrite := os.WriteFile(filepath, resp.AudioContent, defaultFileMode); errWrite != nil {
		log.Errorf("Could not create a result audio file. err: %v", errWrite)
		return errWrite
	}
	log.Debugf("Created a new audio. filename: %s", filepath)

	return nil
}

// getVoiceName returns voicename of the given language and gender
func (h *audioHandler) getVoiceName(lang string, gender tts.Gender) string {
	mapVoiceName := map[string]string{
		"en-US:" + tts.GenderFemale: "en-US-Standard-C",
	}

	tmp := fmt.Sprintf("%s:%s", lang, gender)
	res, ok := mapVoiceName[tmp]
	if !ok {
		return ""
	}

	return res
}
