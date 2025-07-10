package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_elevenlabs.go -source elevenlabs.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"net"
	"strings"
	"time"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"net/url"
)

type ElevenlabsMessage struct {
	Text                 string `json:"text"`
	TryTriggerGeneration bool   `json:"try_trigger_generation"`
	Finalize             bool   `json:"finalize"`
}

type ElevenlabsResponse struct {
	Audio   string `json:"audio,omitempty"`   // Base64 encoded audio data
	IsFinal bool   `json:"isFinal,omitempty"` // Indicates if this is the final message in the stream
	Status  string `json:"status,omitempty"`  // Status message from the server
	Error   string `json:"error,omitempty"`   // Error message if any
}

const (
	defaultElevenlabsHost    = "api.elevenlabs.io"
	defaultElevenlabsTTSPath = "/v1/text-to-speech/%s/stream-input"

	defaultElevenlabsVoiceID      = "EXAVITQu4vr4xnSDxMaL"   // Default voice ID for ElevenLabs
	defaultElevenlabsModelID      = "eleven_multilingual_v2" // Default model ID for ElevenLabs
	defaultConvertSampleRate      = 8000                     // Default sample rate for conversion to 8kHz. This must not be changed as it is the minimum sample rate for audiosocket.
	defaultElevenlabsOutputFormat = "pcm_16000"              // Default output format for ElevenLabs. PCM (S16LE - Signed 16-bit Little Endian), Sample rate: 16kHz, Bit depth: 16-bit as it's the minimum raw PCM output from ElevenLabs.
)

var (
	// Map of ElevenLabs output formats to their corresponding sample rates.
	// https://elevenlabs.io/docs/capabilities/text-to-speech#supported-formats
	elevenlabsFormatToRate = map[string]int{
		"pcm_16000": 16000,
		"pcm_24000": 24000,
		"pcm_48000": 48000,
	}
)

var elevenlabsVoiceIDMap = map[string]string{
	// English
	"english_male":    "21m00Tcm4TlvDq8ikWAM", // Adam
	"english_female":  "EXAVITQu4vr4xnSDxMaL", // Rachel
	"english_neutral": "EXAVITQu4vr4xnSDxMaL", // Rachel

	// Japanese
	"japanese_male":    "21m00Tcm4TlvDq8ikWAM", // use Adam if no male Japanese premade
	"japanese_female":  "yoZ06aMxZJJ28mfd3POQ", // Takumi
	"japanese_neutral": "yoZ06aMxZJJ28mfd3POQ", // Takumi

	// Chinese
	"chinese_male":    "21m00Tcm4TlvDq8ikWAM",
	"chinese_female":  "EXAVITQu4vr4xnSDxMaL",
	"chinese_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Chinese

	// German
	"german_male":    "21m00Tcm4TlvDq8ikWAM",
	"german_female":  "EXAVITQu4vr4xnSDxMaL",
	"german_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral German

	// French
	"french_male":    "21m00Tcm4TlvDq8ikWAM",
	"french_female":  "EXAVITQu4vr4xnSDxMaL",
	"french_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral French

	// Hindi
	"hindi_male":    "21m00Tcm4TlvDq8ikWAM",
	"hindi_female":  "EXAVITQu4vr4xnSDxMaL",
	"hindi_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Hindi

	// Korean
	"korean_male":    "21m00Tcm4TlvDq8ikWAM",
	"korean_female":  "EXAVITQu4vr4xnSDxMaL",
	"korean_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Korean

	// Italian
	"italian_male":    "21m00Tcm4TlvDq8ikWAM",
	"italian_female":  "EXAVITQu4vr4xnSDxMaL",
	"italian_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Italian

	// Spanish (Spain)
	"spanish_male":    "21m00Tcm4TlvDq8ikWAM",
	"spanish_female":  "EXAVITQu4vr4xnSDxMaL",
	"spanish_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Spanish

	// Portuguese (Brazil)
	"portuguese_male":    "21m00Tcm4TlvDq8ikWAM",
	"portuguese_female":  "EXAVITQu4vr4xnSDxMaL",
	"portuguese_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Portuguese

	// Dutch
	"dutch_male":    "21m00Tcm4TlvDq8ikWAM",
	"dutch_female":  "EXAVITQu4vr4xnSDxMaL",
	"dutch_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Dutch

	// Russian
	"russian_male":    "21m00Tcm4TlvDq8ikWAM",
	"russian_female":  "EXAVITQu4vr4xnSDxMaL",
	"russian_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Russian

	// Arabic
	"arabic_male":    "21m00Tcm4TlvDq8ikWAM",
	"arabic_female":  "EXAVITQu4vr4xnSDxMaL",
	"arabic_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Arabic

	// Polish
	"polish_male":    "21m00Tcm4TlvDq8ikWAM",
	"polish_female":  "EXAVITQu4vr4xnSDxMaL",
	"polish_neutral": "EXAVITQu4vr4xnSDxMaL", // Use Rachel for neutral Polish
}

type elevenlabsHandler struct {
	notifyHandler notifyhandler.NotifyHandler

	apiKey string
}

func NewElevenlabsHandler(notifyHandler notifyhandler.NotifyHandler, apiKey string) streamer {
	return &elevenlabsHandler{
		notifyHandler: notifyHandler,

		apiKey: apiKey,
	}
}

func (h *elevenlabsHandler) Run(ctx context.Context, st *streaming.Streaming, conn net.Conn) error {
	connElevenlabs, err := h.initConn(ctx, st)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize ElevenLabs WebSocket connection")
	}
	defer connElevenlabs.Close()

	st.Vendor = streaming.VendorElevenlabs
	st.ConnVendor = connElevenlabs

	go h.runKeepAlive(ctx, st)
	go h.handleReceivedMessages(ctx, conn, st)

	<-st.ChanDone

	return nil
}

func (h *elevenlabsHandler) initConn(ctx context.Context, st *streaming.Streaming) (*websocket.Conn, error) {
	voiceID := h.getVoiceID(st.Language, st.Gender)

	// Construct the WebSocket URL for ElevenLabs.
	u := url.URL{
		Scheme: "wss",
		Host:   defaultElevenlabsHost,
		Path:   fmt.Sprintf(defaultElevenlabsTTSPath, voiceID),
	}

	// Add necessary query parameters for the stream.
	q := u.Query()
	q.Set("model_id", defaultElevenlabsModelID)
	q.Set("optimize_streaming_latency", "4")
	q.Set("output_format", defaultElevenlabsOutputFormat)
	u.RawQuery = q.Encode()

	// Set up WebSocket headers, including the API key.
	header := make(map[string][]string)
	header["xi-api-key"] = []string{h.apiKey}

	// Establish the WebSocket connection.
	res, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to ElevenLabs WebSocket at %s", u.String())
	}

	return res, nil
}

func (h *elevenlabsHandler) send(ctx context.Context, st *streaming.Streaming, msg any) error {

	st.ConnVendorLock.Lock()
	defer st.ConnVendorLock.Unlock()

	conn, ok := st.ConnVendor.(*websocket.Conn)
	if !ok || conn == nil {
		return fmt.Errorf("the ConnVendor is not a *websocket.Conn or is nil")
	}

	return conn.WriteJSON(msg)
}

func (h *elevenlabsHandler) AddText(ctx context.Context, st *streaming.Streaming, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AddText",
		"streaming_id":   st.ID,
		"reference_id":   st.ReferenceID,
		"reference_type": st.ReferenceType,
		"text":           text,
	})

	message := ElevenlabsMessage{
		Text:                 text,
		TryTriggerGeneration: true, // Suggests to the API to start generation if enough text is buffered.
		Finalize:             true,
	}

	log.Debugf("Sending message to ElevenLabs. text: %s", message.Text)
	return h.send(ctx, st, message)
}

func (h *elevenlabsHandler) Finish(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AddText",
		"streaming_id":   st.ID,
		"reference_id":   st.ReferenceID,
		"reference_type": st.ReferenceType,
	})

	message := ElevenlabsMessage{
		Text:                 "", // Empty text signifies the end of input.
		TryTriggerGeneration: true,
		Finalize:             true, // Explicitly tells the API to finalize generation.
	}

	log.Debugf("Sending finalize message to ElevenLabs for streaming ID: %s", st.ID)
	return h.send(ctx, st, message)
}

func (h *elevenlabsHandler) handleReceivedMessages(ctx context.Context, connAst net.Conn, st *streaming.Streaming) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "handleReceivedMessages",
		"streaming_id":   st.ID,
		"reference_id":   st.ReferenceID,
		"reference_type": st.ReferenceType,
	})

	defer func() {
		st.ChanDone <- true
		log.Debugf("handleWebSocketMessages goroutine signaled done.")
	}()

	conn, ok := st.ConnVendor.(*websocket.Conn)
	if !ok || conn == nil {
		return
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("WebSocket closed normally by server or client. Exiting handleWebSocketMessages.")
			} else {
				log.Errorf("Error reading websocket message: %v. Exiting handleWebSocketMessages.", err)
			}
			return
		}
		log.Debugf("Received WebSocket message (type: %d, size: %d bytes)", messageType, len(message))

		var response ElevenlabsResponse
		if errUnmarshal := json.Unmarshal(message, &response); errUnmarshal != nil {
			log.Errorf("Error parsing response: %v. Message: %s", errUnmarshal, string(message))
			continue
		}

		// Process audio data if present.
		if response.Audio != "" {
			log.WithField("audio", response.Audio).Debugf("Received audio chunk for streaming ID: %s", st.ID)

			decodedAudio, decodeErr := base64.StdEncoding.DecodeString(response.Audio)
			if decodeErr != nil {
				log.Errorf("Could not decode base64 audio data: %v. Message: %s", decodeErr, response.Audio)
				return
			}
			log.Debugf("Decoded audio chunk size before processing: %d bytes.", len(decodedAudio))

			data, errProcess := h.convertAndWrapPCMData(defaultElevenlabsOutputFormat, decodedAudio)
			if errProcess != nil {
				log.Errorf("Could not process PCM data: %v. Message: %s", errProcess, response.Audio)
				return
			}
			log.Debugf("Processed audio chunk of size %d bytes.", len(data))

			// send it to the asterisk connection.
			// TTS play!
			n, err := connAst.Write(data)
			if err != nil {
				log.Errorf("Could not write data to asterisk connection: %v", err)
				return
			}

			log.Debugf("Wrote %d bytes to asterisk connection", n)
		}

		// Check for the 'isFinal' flag, which indicates the end of audio generation.
		if response.IsFinal {
			log.Println("Received final message for current generation.")
			h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingFinished, st)
		}

		// Log other control messages like 'status'.
		if response.Status != "" {
			log.Debugf("Status: %s", response.Status)
		}

		// Log any errors reported by the server.
		if response.Error != "" {
			log.Debugf("Error from server: %s", response.Error)
		}
	}
}

// convertAndWrapPCMData converts raw PCM data with the given input format into
// audiosocket-wrapped 16-bit PCM bytes suitable for transmission.
//
// inputFormat: the audio format string (must exist in elevenlabsFormatToRate map)
// data: raw PCM data bytes; must have even length for 16-bit samples.
//
// Returns wrapped PCM bytes or an error on invalid input or processing failure.
func (h *elevenlabsHandler) convertAndWrapPCMData(inputFormat string, data []byte) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "convertAndWrapPCMData",
		"input_format": inputFormat,
		"data_length":  len(data),
	})

	if len(data)%2 != 0 {
		return nil, fmt.Errorf("PCM data length must be even for 16-bit samples (received %d bytes)", len(data))
	}

	// Parse sample rate from format string
	inputRate, ok := elevenlabsFormatToRate[inputFormat]
	if !ok {
		return nil, fmt.Errorf("unsupported input format: %s", inputFormat)
	}

	samples, err := h.getDataSamples(inputRate, data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get samples for format %s", inputFormat)
	}

	res := audiosocket.SlinMessage(samples)

	log.Debugf("Converted and wrapped PCM data. total_len: %d, content_length: %d, kind: %x", len(res), res.ContentLength(), res.Kind())

	// res, err := audiosocketWrapDataPCM16Bit(samples)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "failed to wrap data for audiosocket")
	// }

	return res, nil
}

func (h *elevenlabsHandler) getVoiceID(language string, gender streaming.Gender) string {

	baseLang := strings.ToLower(strings.SplitN(language, "_", 2)[0])
	tmpGender := strings.ToLower(string(gender))
	key := fmt.Sprintf("%s_%s", baseLang, tmpGender)

	if id, ok := elevenlabsVoiceIDMap[key]; ok {
		return id
	}

	// fallback to neutral
	neutralKey := fmt.Sprintf("%s_neutral", baseLang)
	if id, ok := elevenlabsVoiceIDMap[neutralKey]; ok {
		return id
	}

	return defaultElevenlabsVoiceID
}

// getDataSamples processes 16-bit PCM data with the given inputRate sample rate.
// If inputRate equals defaultConvertSampleRate, it returns data as is.
// If inputRate is an integer multiple of defaultConvertSampleRate, it downsamples accordingly.
// Otherwise, it returns an error because only integer downsampling is supported.
func (h *elevenlabsHandler) getDataSamples(inputRate int, data []byte) ([]byte, error) {
	if inputRate == defaultConvertSampleRate {
		// No conversion needed
		return data, nil
	}

	if inputRate%defaultConvertSampleRate != 0 {
		return nil, fmt.Errorf("cannot convert %d Hz to %d Hz: only integer downsampling supported", inputRate, defaultConvertSampleRate)
	}

	factor := inputRate / defaultConvertSampleRate
	res := make([]byte, 0, len(data)/factor)

	// Downsample by selecting every 'factor'-th sample (2 bytes per sample)
	for i := 0; i+2*factor-1 < len(data); i += 2 * factor {
		res = append(res, data[i], data[i+1])
	}

	return res, nil
}

func (h *elevenlabsHandler) runKeepAlive(ctx context.Context, st *streaming.Streaming) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runKeepAlive",
		"streaming_id": st.ID,
	})

	ticker := time.NewTicker(time.Second * 10) // Use configurable interval
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("Keep-alive stopped")
			return
		case <-ticker.C:

			if errText := h.AddText(ctx, st, " "); errText != nil {
				// consider connection has closed.
				return
			}
		}
	}
}
