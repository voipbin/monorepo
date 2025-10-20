package audiohandler

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"monorepo/bin-tts-manager/models/tts"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultAWSRegion = "eu-central-1"
)

func awsGetClient(accessKey string, secretKey string) (*polly.Client, error) {
	// Create AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(defaultAWSRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Initialize Polly client
	res := polly.NewFromConfig(cfg)
	return res, nil
}

func (h *audioHandler) awsAudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender, filepath string) error {
	log := logrus.WithField("func", "awsAudioCreate")

	voiceID := h.awsGetVoiceID(lang, gender)
	input := &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		TextType:     types.TextTypeSsml,
		OutputFormat: types.OutputFormatPcm,
		VoiceId:      voiceID,
		SampleRate:   aws.String(strconv.Itoa(int(defaultSampleRate))),
	}

	start := time.Now()

	resp, err := h.awsClient.SynthesizeSpeech(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to synthesize speech: %w", err)
	}

	if errSave := h.savePCMAsWAV(resp.AudioStream, filepath, int(defaultSampleRate), defaultChannelNum); errSave != nil {
		return fmt.Errorf("failed to save PCM as WAV: %w", errSave)
	}

	elapsed := time.Since(start)
	log.Debugf("SynthesizeSpeech took %s", elapsed)

	return nil
}

func (h *audioHandler) awsGetVoiceID(lang string, gender tts.Gender) types.VoiceId {
	mapVoiceName := map[string]types.VoiceId{
		"en-US:" + string(tts.GenderFemale):  types.VoiceIdJoanna,
		"en-US:" + string(tts.GenderMale):    types.VoiceIdMatthew,
		"en-US:" + string(tts.GenderNeutral): types.VoiceIdJoey,

		"en-GB:" + string(tts.GenderFemale):  types.VoiceIdAmy,
		"en-GB:" + string(tts.GenderMale):    types.VoiceIdBrian,
		"en-GB:" + string(tts.GenderNeutral): types.VoiceIdEmma,

		"de-DE:" + string(tts.GenderFemale):  types.VoiceIdMarlene,
		"de-DE:" + string(tts.GenderMale):    types.VoiceIdHans,
		"de-DE:" + string(tts.GenderNeutral): types.VoiceIdVicki,

		"fr-FR:" + string(tts.GenderFemale):  types.VoiceIdCeline,
		"fr-FR:" + string(tts.GenderMale):    types.VoiceIdMathieu,
		"fr-FR:" + string(tts.GenderNeutral): types.VoiceIdLea,

		"es-ES:" + string(tts.GenderFemale):  types.VoiceIdConchita,
		"es-ES:" + string(tts.GenderMale):    types.VoiceIdEnrique,
		"es-ES:" + string(tts.GenderNeutral): types.VoiceIdLucia,

		"it-IT:" + string(tts.GenderFemale):  types.VoiceIdCarla,
		"it-IT:" + string(tts.GenderMale):    types.VoiceIdGiorgio,
		"it-IT:" + string(tts.GenderNeutral): types.VoiceIdBianca,

		"ja-JP:" + string(tts.GenderFemale): types.VoiceIdMizuki,
		"ja-JP:" + string(tts.GenderMale):   types.VoiceIdTakumi,

		"ko-KR:" + string(tts.GenderFemale):  types.VoiceIdSeoyeon,
		"ko-KR:" + string(tts.GenderNeutral): types.VoiceIdJihye,

		"pt-BR:" + string(tts.GenderFemale):  types.VoiceIdCamila,
		"pt-BR:" + string(tts.GenderMale):    types.VoiceIdRicardo,
		"pt-BR:" + string(tts.GenderNeutral): types.VoiceIdCamila,

		"ru-RU:" + string(tts.GenderFemale):  types.VoiceIdTatyana,
		"ru-RU:" + string(tts.GenderMale):    types.VoiceIdMaxim,
		"ru-RU:" + string(tts.GenderNeutral): types.VoiceIdTatyana,

		"zh-CN:" + string(tts.GenderFemale):  types.VoiceIdZhiyu,
		"zh-CN:" + string(tts.GenderMale):    types.VoiceIdZhiyu,
		"zh-CN:" + string(tts.GenderNeutral): types.VoiceIdZhiyu,
	}

	tmp := fmt.Sprintf("%s:%s", lang, gender)
	res, ok := mapVoiceName[tmp]
	if !ok {
		return ""
	}

	return res
}

// savePCMAsWAV writes PCM data to a WAV file with the correct header
func (h *audioHandler) savePCMAsWAV(pcmStream io.Reader, filename string, sampleRate, numChannels int) error {
	// Open file for writing
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Convert sample rate & channels to WAV format parameters
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * (bitsPerSample / 8)
	blockAlign := numChannels * (bitsPerSample / 8)

	// Write WAV header with proper error checks
	header := []interface{}{
		[]byte("RIFF"),        // ChunkID
		uint32(0),             // ChunkSize (placeholder)
		[]byte("WAVE"),        // Format
		[]byte("fmt "),        // Subchunk1ID
		uint32(16),            // Subchunk1Size (PCM)
		uint16(1),             // AudioFormat (PCM = 1)
		uint16(numChannels),   // NumChannels
		uint32(sampleRate),    // SampleRate
		uint32(byteRate),      // ByteRate
		uint16(blockAlign),    // BlockAlign
		uint16(bitsPerSample), // BitsPerSample
		[]byte("data"),        // Subchunk2ID
		uint32(0),             // Subchunk2Size (placeholder)
	}

	for _, v := range header {
		if err := binary.Write(file, binary.LittleEndian, v); err != nil {
			return fmt.Errorf("failed to write WAV header: %w", err)
		}
	}

	// Write PCM data
	dataSize, err := io.Copy(file, pcmStream)
	if err != nil {
		return fmt.Errorf("failed to write PCM data: %w", err)
	}

	// Update file sizes in WAV header
	if _, err := file.Seek(4, 0); err != nil {
		return fmt.Errorf("failed to seek to ChunkSize: %w", err)
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(36+dataSize)); err != nil {
		return fmt.Errorf("failed to update ChunkSize: %w", err)
	}

	if _, err := file.Seek(40, 0); err != nil {
		return fmt.Errorf("failed to seek to Subchunk2Size: %w", err)
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(dataSize)); err != nil {
		return fmt.Errorf("failed to update Subchunk2Size: %w", err)
	}

	return nil
}
