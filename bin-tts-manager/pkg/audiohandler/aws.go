package audiohandler

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
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

func (h *audioHandler) awsAudioCreate(ctx context.Context, callID uuid.UUID, text string, lang string, voiceID string, filepath string) error {
	log := logrus.WithField("func", "awsAudioCreate")

	voice := h.awsGetDefaultVoiceID(lang)
	if voiceID != "" {
		voice = types.VoiceId(voiceID)
	}
	if voice == "" {
		return fmt.Errorf("no default voice available for language %q and no voice_id provided", lang)
	}
	input := &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		TextType:     types.TextTypeSsml,
		OutputFormat: types.OutputFormatPcm,
		VoiceId:      voice,
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

// awsGetDefaultVoiceID returns default voice ID for the given language
func (h *audioHandler) awsGetDefaultVoiceID(lang string) types.VoiceId {
	defaultVoices := map[string]types.VoiceId{
		"en-US": types.VoiceIdJoanna,
		"en-GB": types.VoiceIdAmy,
		"de-DE": types.VoiceIdMarlene,
		"fr-FR": types.VoiceIdCeline,
		"es-ES": types.VoiceIdConchita,
		"it-IT": types.VoiceIdCarla,
		"ja-JP": types.VoiceIdMizuki,
		"ko-KR": types.VoiceIdSeoyeon,
		"pt-BR": types.VoiceIdCamila,
		"ru-RU": types.VoiceIdTatyana,
		"zh-CN": types.VoiceIdZhiyu,
	}

	res, ok := defaultVoices[lang]
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
