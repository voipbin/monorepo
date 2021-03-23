package audiohandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package audiohandler -destination ./mock_audiohandler_audiohandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/ioutil"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

// AudioHandler intreface for audio handler
type AudioHandler interface {
	AudioCreate(text, lang, gender, filename string) error
}

type audioHandler struct {
	client *texttospeech.Client
}

// NewAudioHandler create AudioHandler
func NewAudioHandler(credentialPath string) AudioHandler {
	ctx := context.Background()

	// create client
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &audioHandler{
		client: client,
	}

	return h
}

// Init initialize the Audio handler
func (h *audioHandler) Init() {
	return
}

func (h *audioHandler) AudioCreate(text, lang, gender, filename string) error {

	ssmlGender := texttospeechpb.SsmlVoiceGender_NEUTRAL
	switch gender {
	case "male":
		ssmlGender = texttospeechpb.SsmlVoiceGender_MALE
		break

	case "female":
		ssmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
		break
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
	ctx := context.Background()
	resp, err := h.client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		logrus.Errorf("Could not get a correct response. text: %s, lang: %s, ssmlGender: %v, err: %v", text, lang, ssmlGender, err)
		return err
	}

	// create audio
	if err := ioutil.WriteFile(filename, resp.AudioContent, 0644); err != nil {
		logrus.Errorf("Could not create a result audio file. err: %v", err)
		return err
	}

	return nil
}
