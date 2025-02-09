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
		SampleRate:   aws.String(strconv.Itoa(int(defaultSampleRate))),
	}

	start := time.Now()

	// Generate speech
	resp, err := h.awsClient.SynthesizeSpeech(input)
	if err != nil {
		// log.Fatalf("Failed to synthesize speech: %v", err)
		return fmt.Errorf("failed to synthesize speech: %w", err)
	}

	if errSave := h.savePCMAsWAV(resp.AudioStream, filepath, int(defaultSampleRate), 1); errSave != nil {
		return fmt.Errorf("failed to save PCM as WAV: %w", errSave)
	}

	// audioData, err := io.ReadAll(resp.AudioStream)
	// if err != nil {
	// 	return fmt.Errorf("failed to read audio stream: %v", err)
	// }

	// if err := os.WriteFile(filepath, audioData, 0644); err != nil {
	// 	log.Fatalf("Failed to write audio file: %v", err)
	// }

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

// savePCMAsWAV writes PCM data to a WAV file with the correct header
func (h *audioHandler) savePCMAsWAV(pcmStream io.Reader, filename string, sampleRate, numChannels int) error {
	// Open file for writing
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Convert sample rate & channels to WAV format parameters
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * (bitsPerSample / 8)
	blockAlign := numChannels * (bitsPerSample / 8)

	// // Write WAV header
	// err = binary.Write(file, binary.LittleEndian, []byte("RIFF"))        // ChunkID
	// err = binary.Write(file, binary.LittleEndian, uint32(0))             // ChunkSize (placeholder)
	// err = binary.Write(file, binary.LittleEndian, []byte("WAVE"))        // Format
	// err = binary.Write(file, binary.LittleEndian, []byte("fmt "))        // Subchunk1ID
	// err = binary.Write(file, binary.LittleEndian, uint32(16))            // Subchunk1Size (PCM)
	// err = binary.Write(file, binary.LittleEndian, uint16(1))             // AudioFormat (PCM = 1)
	// err = binary.Write(file, binary.LittleEndian, uint16(numChannels))   // NumChannels
	// err = binary.Write(file, binary.LittleEndian, uint32(sampleRate))    // SampleRate
	// err = binary.Write(file, binary.LittleEndian, uint32(byteRate))      // ByteRate
	// err = binary.Write(file, binary.LittleEndian, uint16(blockAlign))    // BlockAlign
	// err = binary.Write(file, binary.LittleEndian, uint16(bitsPerSample)) // BitsPerSample
	// err = binary.Write(file, binary.LittleEndian, []byte("data"))        // Subchunk2ID
	// err = binary.Write(file, binary.LittleEndian, uint32(0))             // Subchunk2Size (placeholder)
	// if err != nil {
	// 	return fmt.Errorf("failed to write WAV header: %w", err)
	// }

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

	// // Update file sizes in WAV header
	// file.Seek(4, 0)
	// binary.Write(file, binary.LittleEndian, uint32(36+dataSize)) // RIFF ChunkSize
	// file.Seek(40, 0)
	// binary.Write(file, binary.LittleEndian, uint32(dataSize)) // Data ChunkSize

	return nil
}
