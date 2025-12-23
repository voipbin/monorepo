package audiohandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"monorepo/bin-tts-manager/models/tts"
	"os"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	defaultGCPEndpoint = "eu-texttospeech.googleapis.com:443"
)

func gcpGetClient(ctx context.Context, credentialBase64 string) (*texttospeech.Client, error) {
	decodedCredential, err := base64.StdEncoding.DecodeString(credentialBase64)
	if err != nil {
		return nil, errors.Wrapf(err, "could not decode the base64 credential")
	}

	keepAliveParams := keepalive.ClientParameters{
		Time:                30 * time.Second, // Ping every 30 seconds
		Timeout:             10 * time.Second, // Wait 10 seconds for response
		PermitWithoutStream: true,             // Send pings even if there are no active streams
	}

	creds, err := google.CredentialsFromJSON(ctx, decodedCredential, texttospeech.DefaultAuthScopes()...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process credentials")
	}

	res, err := texttospeech.NewClient(
		ctx,
		option.WithCredentials(creds),
		option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepAliveParams)),
		option.WithEndpoint(defaultGCPEndpoint),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new client")
	}

	return res, nil
}

// AudioCreate Creates tts audio
func (h *audioHandler) gcpAudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "gcpAudioCreate",
		"call_id": callID,
	})
	log.WithField("text", text).Debugf("Creating a new audio. lang: %s, gender: %s, filepath: %s", lang, gender, filepath)

	voiceName := h.gcpGetVoiceName(lang, gender)
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

	start := time.Now()

	log.Debugf("Sending speech request. language_code: %s, gender: %d, name: %s", req.Voice.LanguageCode, req.Voice.SsmlGender, voiceName)
	resp, err := h.gcpClient.SynthesizeSpeech(ctx, &req)
	if err != nil {
		log.Errorf("Could not get a correct response. text: %s, lang: %s, ssmlGender: %v, err: %v", text, lang, ssmlGender, err)
		return err
	}

	// create audio
	log.Debugf("Writing audio content to file. filepath: %s", filepath)
	if errWrite := os.WriteFile(filepath, resp.AudioContent, defaultFileMode); errWrite != nil {
		log.Errorf("Could not create a result audio file. err: %v", errWrite)
		return errWrite
	}
	log.Debugf("Created a new audio. filename: %s", filepath)

	elapsed := time.Since(start)
	log.Debugf("SynthesizeSpeech took %s", elapsed)

	return nil
}

// gcpGetVoiceName returns voicename of the given language and gender
func (h *audioHandler) gcpGetVoiceName(lang string, gender tts.Gender) string {
	mapVoiceName := map[string]string{
		"en-US:" + string(tts.GenderFemale):  "en-US-Wavenet-F",
		"en-US:" + string(tts.GenderMale):    "en-US-Wavenet-D",
		"en-US:" + string(tts.GenderNeutral): "en-US-Wavenet-A",
		"en-GB:" + string(tts.GenderFemale):  "en-GB-Wavenet-A",
		"en-GB:" + string(tts.GenderMale):    "en-GB-Wavenet-B",
		"en-GB:" + string(tts.GenderNeutral): "en-GB-Wavenet-D",
		"de-DE:" + string(tts.GenderFemale):  "de-DE-Wavenet-F",
		"de-DE:" + string(tts.GenderMale):    "de-DE-Wavenet-D",
		"de-DE:" + string(tts.GenderNeutral): "de-DE-Wavenet-A",
		"fr-FR:" + string(tts.GenderFemale):  "fr-FR-Wavenet-E",
		"fr-FR:" + string(tts.GenderMale):    "fr-FR-Wavenet-B",
		"fr-FR:" + string(tts.GenderNeutral): "fr-FR-Wavenet-A",
		"es-ES:" + string(tts.GenderFemale):  "es-ES-Wavenet-E",
		"es-ES:" + string(tts.GenderMale):    "es-ES-Wavenet-B",
		"es-ES:" + string(tts.GenderNeutral): "es-ES-Wavenet-A",
		"it-IT:" + string(tts.GenderFemale):  "it-IT-Wavenet-E",
		"it-IT:" + string(tts.GenderMale):    "it-IT-Wavenet-B",
		"it-IT:" + string(tts.GenderNeutral): "it-IT-Wavenet-A",
		"ja-JP:" + string(tts.GenderFemale):  "ja-JP-Wavenet-C",
		"ja-JP:" + string(tts.GenderMale):    "ja-JP-Wavenet-B",
		"ko-KR:" + string(tts.GenderFemale):  "ko-KR-Wavenet-C",
		"ko-KR:" + string(tts.GenderNeutral): "ko-KR-Wavenet-A",
	}

	tmp := fmt.Sprintf("%s:%s", lang, gender)
	res, ok := mapVoiceName[tmp]
	if !ok {
		return ""
	}

	return res

}
