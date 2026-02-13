package audiohandler

import (
	"context"
	"os"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	defaultGCPEndpoint = "eu-texttospeech.googleapis.com:443"
)

// gcpGetClient creates a Google Cloud Text-to-Speech client using Application Default Credentials (ADC).
// Callers must ensure the environment is configured for ADC (for example via
// GOOGLE_APPLICATION_CREDENTIALS, workload identity, or in-cluster metadata).
func gcpGetClient(ctx context.Context) (*texttospeech.Client, error) {
	keepAliveParams := keepalive.ClientParameters{
		Time:                30 * time.Second, // Ping every 30 seconds
		Timeout:             10 * time.Second, // Wait 10 seconds for response
		PermitWithoutStream: true,             // Send pings even if there are no active streams
	}

	// We know staticcheck flags this, but the texttospeech client library
	// has not yet been updated to use the new context package.
	//nolint:staticcheck
	res, err := texttospeech.NewClient(
		ctx,
		option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepAliveParams)),
		option.WithEndpoint(defaultGCPEndpoint),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create a new client")
	}

	return res, nil
}

func (h *audioHandler) gcpAudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, voiceID string, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "gcpAudioCreate",
		"call_id": callID,
	})
	log.WithField("text", text).Debugf("Creating a new audio. lang: %s, voice_id: %s, filepath: %s", lang, voiceID, filepath)

	voiceName := voiceID
	if voiceName == "" {
		voiceName = h.gcpGetDefaultVoiceName(lang)
	}
	if voiceName == "" {
		log.Debugf("No default voice for language %q, using GCP auto-selection", lang)
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

		// build the voice request, select the language code and voice name
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: lang,
			Name:         voiceName,
		},

		// select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   defaultAudioEncoding,
			SampleRateHertz: defaultSampleRate,
		},
	}

	start := time.Now()

	log.Debugf("Sending speech request. language_code: %s, name: %s", req.Voice.LanguageCode, voiceName)
	resp, err := h.gcpClient.SynthesizeSpeech(ctx, &req)
	if err != nil {
		log.Errorf("Could not get a correct response. text: %s, lang: %s, voice_name: %s, err: %v", text, lang, voiceName, err)
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

// gcpGetDefaultVoiceName returns default voice name for the given language
func (h *audioHandler) gcpGetDefaultVoiceName(lang string) string {
	defaultVoices := map[string]string{
		"en-US": "en-US-Wavenet-F",
		"en-GB": "en-GB-Wavenet-A",
		"de-DE": "de-DE-Wavenet-F",
		"fr-FR": "fr-FR-Wavenet-E",
		"es-ES": "es-ES-Wavenet-E",
		"it-IT": "it-IT-Wavenet-E",
		"ja-JP": "ja-JP-Wavenet-C",
		"ko-KR": "ko-KR-Wavenet-C",
	}

	res, ok := defaultVoices[lang]
	if !ok {
		return ""
	}

	return res
}
