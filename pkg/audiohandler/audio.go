package audiohandler

import (
	"context"
	"io/ioutil"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

func (h *audioHandler) AudioCreate(ctx context.Context, callID uuid.UUID, text, lang, gender, filename string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AudioCreate",
		"call_id": callID,
	})
	log.WithField("text", text).Debugf("Creating a new audio. lang: %s, gender: %s, filename: %s", lang, gender, filename)

	ssmlGender := texttospeechpb.SsmlVoiceGender_NEUTRAL
	switch gender {
	case "male":
		ssmlGender = texttospeechpb.SsmlVoiceGender_MALE

	case "female":
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
			SsmlGender:   ssmlGender,
		},

		// select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_LINEAR16,
			SampleRateHertz: 8000,
		},
	}

	// send request
	resp, err := h.client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		log.Errorf("Could not get a correct response. text: %s, lang: %s, ssmlGender: %v, err: %v", text, lang, ssmlGender, err)
		return err
	}

	// create audio
	if err := ioutil.WriteFile(filename, resp.AudioContent, 0644); err != nil {
		log.Errorf("Could not create a result audio file. err: %v", err)
		return err
	}
	log.Debugf("Created a new audio. filename: %s", filename)

	return nil
}
