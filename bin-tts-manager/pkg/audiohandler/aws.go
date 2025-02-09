package audiohandler

import (
	"context"
	"fmt"
	"io"
	"monorepo/bin-tts-manager/models/tts"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/polly"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultAWSRegion = "eu-central-1"
)

func awsGetClient(accessKey string, secretKey string) (*polly.Polly, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(defaultAWSRegion),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Initialize Polly client
	res := polly.New(sess)
	return res, nil
}

func (h *audioHandler) awsAudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filepath string) error {
	log := logrus.WithField("func", "awsAudioCreate")

	voiceID := h.awsGetVoiceID(lang, gender)
	input := &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		OutputFormat: aws.String(polly.OutputFormatPcm),
		VoiceId:      aws.String(voiceID),
	}

	start := time.Now()

	// Generate speech
	resp, err := h.awsClient.SynthesizeSpeech(input)
	if err != nil {
		log.Fatalf("Failed to synthesize speech: %v", err)
	}

	audioData, err := io.ReadAll(resp.AudioStream)
	if err != nil {
		return fmt.Errorf("failed to read audio stream: %v", err)
	}

	if err := os.WriteFile(filepath, audioData, 0644); err != nil {
		log.Fatalf("Failed to write audio file: %v", err)
	}

	elapsed := time.Since(start)
	log.Debugf("SynthesizeSpeech took %s", elapsed)

	return nil
}

func (h *audioHandler) awsGetVoiceID(lang string, gender tts.Gender) string {
	mapVoiceName := map[string]string{
		"en-US:" + string(tts.GenderFemale):  "Joanna",
		"en-US:" + string(tts.GenderMale):    "Matthew",
		"en-US:" + string(tts.GenderNeutral): "Joey",

		"en-GB:" + string(tts.GenderFemale):  "Amy",
		"en-GB:" + string(tts.GenderMale):    "Brian",
		"en-GB:" + string(tts.GenderNeutral): "Emma",

		"de-DE:" + string(tts.GenderFemale):  "Marlene",
		"de-DE:" + string(tts.GenderMale):    "Hans",
		"de-DE:" + string(tts.GenderNeutral): "Vicki",

		"fr-FR:" + string(tts.GenderFemale):  "Celine",
		"fr-FR:" + string(tts.GenderMale):    "Mathieu",
		"fr-FR:" + string(tts.GenderNeutral): "Lea",

		"es-ES:" + string(tts.GenderFemale):  "Conchita",
		"es-ES:" + string(tts.GenderMale):    "Enrique",
		"es-ES:" + string(tts.GenderNeutral): "Lucia",

		"it-IT:" + string(tts.GenderFemale):  "Carla",
		"it-IT:" + string(tts.GenderMale):    "Giorgio",
		"it-IT:" + string(tts.GenderNeutral): "Bianca",

		"ja-JP:" + string(tts.GenderFemale): "Mizuki",
		"ja-JP:" + string(tts.GenderMale):   "Takumi",

		"ko-KR:" + string(tts.GenderFemale):  "Seoyeon",
		"ko-KR:" + string(tts.GenderNeutral): "Jisoo",
	}

	tmp := fmt.Sprintf("%s:%s", lang, gender)
	res, ok := mapVoiceName[tmp]
	if !ok {
		return ""
	}

	return res
}
