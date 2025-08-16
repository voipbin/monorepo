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

	// go h.runKeepAlive(ctx, st)
	go h.handleReceivedMessages(ctx, conn, st)

	go h.testSay(ctx, conn, st)

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
		"func":      "AddText",
		"streaming": st,
		"text":      text,
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

func (h *elevenlabsHandler) testSay(ctx context.Context, connAst net.Conn, st *streaming.Streaming) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "testSay",
		"streaming_id": st.ID,
	})

	const chunkSize = 4096

	for {
		select {
		case <-ctx.Done():
			log.Debug("Context done, stopping testSay.")
			return
		default:
			log.Debugf("Sending test audio data to asterisk connection for streaming ID: %s", st.ID)
			decodeSample, err := base64.StdEncoding.DecodeString(testSampleData)
			if err != nil {
				log.Errorf("Failed to decode test sample data: %v", err)
				return
			}

			// data, errProcess := h.convertAndWrapPCMData(defaultElevenlabsOutputFormat, decodeSample)
			// if errProcess != nil {
			// 	log.Errorf("Failed to process PCM data for test audio: %v", errProcess)
			// 	return
			// }
			// log.Debugf("Processed audio chunk of size %d bytes.", len(data))

			// send it to the asterisk connection.
			// TTS play!
			// n, err := connAst.Write(data)
			// if err != nil {
			// 	log.Errorf("Could not write data to asterisk connection: %v", err)
			// 	return
			// }

			total := len(decodeSample)
			sent := 0

			for sent < total {
				end := sent + chunkSize
				if end > total {
					end = total
				}

				sample := decodeSample[sent:end]
				data, err := h.convertAndWrapPCMData(defaultElevenlabsOutputFormat, sample)
				if err != nil {
					log.Errorf("Could not process PCM data for test audio chunk: %v", err)
					return
				}

				_, err = connAst.Write(data)
				if err != nil {
					log.Printf("write failed: %v", err)
					return
				}
				sent += len(sample)
			}

			log.Debugf("Wrote %d bytes to asterisk connection", sent)
		}

		time.Sleep(2 * time.Second) // Sleep for 2 seconds before sending the next test audio chunk.
	}
}

func (h *elevenlabsHandler) handleReceivedMessages(ctx context.Context, connAst net.Conn, st *streaming.Streaming) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "handleReceivedMessages",
		"streaming": st,
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
			// TTS play
			if errWrite := audiosocketWrite(connAst, data); errWrite != nil {
				log.Errorf("Could not write processed audio data to asterisk connection: %v", errWrite)
				return
			}
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

// func (h *elevenlabsHandler) runKeepAlive(ctx context.Context, st *streaming.Streaming) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":         "runKeepAlive",
// 		"streaming_id": st.ID,
// 	})

// 	ticker := time.NewTicker(time.Second * 10) // Use configurable interval
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			log.Debug("Keep-alive stopped")
// 			return
// 		case <-ticker.C:

// 			if errText := h.AddText(ctx, st, " "); errText != nil {
// 				// consider connection has closed.
// 				return
// 			}
// 		}
// 	}
// }

const (
	testSampleData = "//8EAAQAAgABAP/////+//7//f/+//3///8AAP7//v///wEAAQABAAEAAAABAAAAAAABAAEAAQABAAEAAQABAAEAAQACAAIAAgACAAEAAQAAAAEAAQACAAEAAAACAAEAAgABAAEAAQAAAAAAAQABAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAEAAAABAAAAAQAAAAEAAQABAAEAAAABAAEAAQACAAEAAQACAAAAAgABAAEAAQABAAMAAAACAAIAAgADAAMAAgACAAIAAQADAAIAAwACAAIAAwACAAQAAgADAAMAAgACAAIAAwABAAIAAgADAAMAAgACAAMAAgACAAEAAgADAAIAAQABAAIAAgABAAEAAgAAAAEAAAABAAEAAgAAAAAAAgABAAAAAQD//wAAAQAAAAEAAQAAAAAAAQABAAAAAAABAAAAAAAAAP//AAD//wAA/////wEA//8AAP//AAD//////////wAAAAD//wAAAAD/////AAD+////AAD/////AAD//wAA//8AAP////8AAP///////wAAAAD//wAAAAD//wAAAAD//wAAAAAAAAEAAAAAAAAAAAABAAAAAAD/////AAABAAAAAAABAAAAAAAAAAAAAQAAAAEAAAABAAAAAAABAAEAAAABAAAAAQABAAAAAQAAAAIAAAABAAEAAQABAAEAAQAAAAIAAQABAAIAAQACAAEAAgACAAIAAQABAAEAAQADAAEAAQACAAEAAQACAAIAAQADAAEAAgABAAIAAgABAAIAAAABAAIAAQADAAEAAQADAAIAAgACAAMAAQACAAIAAQACAAEAAAABAAAAAgACAAEAAQACAAEAAgABAAIAAQACAAIAAgACAAEAAgAAAAEAAgACAAAAAgACAAEAAgAAAAEAAQAAAAEAAAABAAEAAQACAAEAAgABAAMAAwADAAMAAwACAAIAAgACAAEAAgAAAP/////9/////f8AAAAA//8AAAAAAAD+//////8AAAAA//8AAAEA//8BAP//AAAAAAAAAAAAAP////8AAAAAAAAAAAEAAAABAAAAAQABAAAAAAAAAP//AAAAAP//AAAAAAEAAAABAAAAAAABAAAAAAAAAAEAAQAAAAEAAAAAAAAAAAAAAAAAAAD//wAAAQABAAAAAAABAAEAAQAAAAEAAQAAAAIAAQABAAEAAQABAAEAAgABAAEAAQABAAEAAgAAAAEAAQABAAEAAQABAAEAAQABAAIAAQABAAEAAQABAAEAAgACAAEAAAAAAAIAAQABAAAAAAACAAAAAQACAAAAAQABAAEAAQABAAEAAQABAAEAAgABAAIAAQACAAAAAQAAAAEAAQABAAEAAQABAAIAAQABAAIAAQABAAAAAgAAAAAAAgAAAAIAAAABAAEAAAAAAAAAAQAAAAAA//8AAAAAAAAAAAAAAAAAAP//AAD//wAA//8AAAAAAAD//wAAAAAAAAAAAQD///////8AAAAA//8BAAAA/////wAA//8AAAAA/////wAAAAD/////AAAAAP//AAD//wAAAQAAAP//AQAAAAAAAAAAAAAAAQABAP//AAABAAAAAAD//wEAAQD+/wAAAQAAAAEAAQAAAAEAAAABAAAAAQAAAAEAAQABAAAAAQABAAAAAQAAAAEAAQACAAEAAQACAAAAAQAAAAEAAgABAAEAAQABAAEAAQABAAEAAQACAAIAAQACAAIAAQABAAEAAgABAAIAAgABAAEAAgACAAIAAAABAAIAAgACAAEAAgACAAIAAgABAAMAAQABAAIAAQACAAEAAgADAAEAAgABAAEAAQABAAIAAQABAAIAAgACAAIAAgACAAEAAAAAAAEAAgACAAEAAQACAAEAAQABAAAAAQABAAAAAgAAAAIAAQAAAAIAAAABAAEAAQABAAMAAgADAAEAAwACAAIAAgABAAMAAgADAAQAAQAAAAAAAQAAAAAAAAAAAAEAAAAAAP///v////////////7/////////AAAAAP7//////wAA/v///wAA/////wAA//////////////////////7//////////////wEAAAAAAP//AAAAAP//AAD//wAAAQAAAAAAAQABAAEAAQABAAAAAQABAAEAAQACAAEAAQACAAEAAQACAAEAAgACAAIAAgACAAMAAgACAAIAAQACAAMAAQADAAIAAwADAAMAAwADAAIAAwADAAMABAACAAIAAwAEAAMAAwADAAIAAwAEAAMAAwACAAMAAwADAAIAAgADAAMAAgADAAIAAgACAAIAAwACAAIAAgACAAEAAgACAAEAAgABAAIAAgADAAIAAgACAAIAAQACAAAAAQACAAEAAQABAAEAAQABAAAAAQABAAAAAAAAAAAAAQABAAAAAQAAAAAA//8AAAAA//8AAAAAAAAAAAAA////////////////////////////////AAD+///////+/////v////7///////3//f/////////+//7//v/+/////f///////v/+///////+/////////////v/+//7//v//////////////AAD/////AAD/////AAD/////AAAAAP//AAD/////AAD//wAAAAAAAAEAAQABAAEAAAABAAAAAgAAAAIAAQABAAEAAQACAAIAAQACAAIAAgADAAIAAwADAAIAAgADAAMAAwADAAQAAwADAAIAAwADAAMABAADAAMABAAEAAUABQADAAMAAwADAAQABAACAAQAAwADAAMABAAEAAIABAADAAQAAwADAAMAAwAEAAMAAwADAAQAAwACAAIAAQADAAIAAgACAAIAAgABAAEAAQABAAEAAQAAAAAAAAAAAAEAAAABAAAAAAAAAAAA/////wAA////////AAAAAAEAAQAAAAEAAAAAAAEAAQAAAAEAAAAAAAEAAAAAAAAA//8AAP7//////wAAAAD+//3//f/9//3//f/9//7//f/+//z//f/+//7//v/9//7////+//7//v/+//3///////7///////7/////////////////AAD+//////8AAAAA//8AAAAAAAABAAEAAAABAAIAAQABAAEAAQABAAMAAgADAAIAAQADAAIAAwACAAMAAwADAAMAAwADAAIAAwADAAMAAwADAAQAAwAEAAQABAAEAAQABAAFAAUABAAFAAUABAAFAAUABAAEAAYABQAFAAYABQAFAAUABQAFAAYABAAFAAUABQAFAAUABQAFAAUABAAFAAQABAAFAAQABAAFAAQABQAEAAQABAAEAAQAAwADAAQAAwADAAMAAwADAAEAAgACAAEAAgACAAEAAgABAAEAAQAAAAAAAQAAAP//AAD//wAA/////////v////7////////////9//7////+/////f/+//3//P/+//3//f/8//3//v/9//z//f/8//3//P/9//v//P/8//z/+//8//v//P/9//3//f/8//z//P/9//z//f/9//3//P/8//3//f/9//3//v/9//3//P/9//3//v/9//z//v/9/////v/+/////v/+//////////////8AAP///////wAAAAABAAAAAQABAAEAAAABAAEAAwACAAIAAgACAAIAAwADAAIAAwADAAQAAwAEAAMABAACAAQABAADAAQABAAGAAUABQAGAAUABQAFAAYABQAGAAUABQAFAAYABwAGAAUABgAHAAYABwAGAAYABwAGAAYABgAHAAYABQAGAAYABwAFAAYABgAGAAYABQAEAAUABAAFAAQABAAFAAUABQACAAQABAADAAMAAwADAAMAAgADAAIAAgACAAEAAgAAAAIAAAD//wEAAAABAAAA//////7//v/9//7////////////+//7//v/+/////v/+//7/+//8//3//f/8//z//f/8//3//f/8//3//P/7//n/+v/5//n/+f/6//n/+f/5//n/+v/5//n/+v/6//r/+v/6//r/+v/6//v/+//6//v/+//7//v/+//8//z//P/7//3//f/9//3//f/9//3//f/+//7//////wAA//8AAAAAAAAAAAEAAgABAAIAAgACAAIAAgACAAIABAADAAIABAAEAAMABQAFAAUABQAEAAYABgAFAAYABgAHAAYABwAHAAcABwAIAAgACAAIAAgACAAIAAcACAAIAAcACAAJAAgACAAIAAcACQAJAAgACAAHAAgABwAIAAcACAAIAAcABwAHAAcABgAHAAcABwAHAAYABgAFAAYABgAGAAUABQAFAAYABQAEAAUABAADAAMAAwACAAMAAwACAAIAAQABAAIAAgAAAAEAAAAAAAAA///////////+/////v/+//3//f/9//3//f/9//z//P/7//v//f/6//v/+v/6//v/+v/6//n/+v/6//n/+v/6//n/+P/5//n/+P/4//j/+f/4//n/+P/5//r/+P/4//f/+P/4//j/+f/5//j/+f/6//j/+f/6//n/+v/5//n/+v/6//n/+v/7//r/+//8//z//P/8//3//f/9//7//f/9//7////+//7////+/wAAAAAAAAAAAQABAAEAAQACAAMAAwACAAMAAwADAAQABAAEAAUABQAFAAYABgAGAAcABwAHAAgABwAJAAgACAAJAAkACgAKAAoACQAJAAkACgAKAAoACgAKAAsACgALAAsACgAKAAoACwAKAAsACQAKAAoACQAKAAoACgAJAAoACQAJAAkACAAIAAcABwAHAAYABgAFAAUABgAEAAQABAADAAMAAgACAAIAAgABAAAA//8AAAAA//////7////+/////f/9//z//P/8//v//P/6//n/+v/5//n/+v/7//v/+v/5//r/+f/6//z/+v/6//n/+v/6//r/+P/3//b/9f/0//b/9v/1//X/9v/2//X/9f/1//b/9v/0//X/9v/1//f/9f/4//f/9v/3//j/9//4//j/+P/4//n/+f/5//n/+v/6//r//P/7//z//f/8//3//P/8//3//f/9//3//f/9//7//v/+//////8BAP//AAAAAAAAAQABAAEAAgADAAIAAwADAAQAAwADAAQABQAFAAUABgAGAAcACAAJAAoACAAJAAkACwAMAAwADQAMAA4ADwAPAA8AEAARABAAEAAQAA8ADwAQABAAEAAQABAAEQAQABEAEQARABEAEQAQABAAEAAQAA8ADwAOAA4ADQAMAAwACwAMAAwADAAKAAoACQAIAAcABwAHAAYABgAGAAUABQACAAUABAABAAIA/////wEA//8AAP///P/8//z/+//7//n/+P/4//j/9//3//j/9//2//P/9P/x//L/8f/x//L/7//x/+7/8v/x//T/9//0//L/8v/u/+3/7P/u//D/7f/u/+3/7f/u/+3/7P/r/+3/6//v/+//7//x//T/9v/6//j/9v/0//H/8v/u//H/8f/t/+3/6//t//T/+v/7//z/+v/4//b/9P/z//P/9v/z//L/9f/9/wEAAAAAAAIAAgABAAEAAQAFAAYAAwD8//P/8//7/wcAEAAXABcAEgACAPj/+/8SADMALwAcAAAA5v/k//D/AQAUABcAGgAOAPf/6//y/wUAFgAXAP//5v/p/w4AOABZAFwAUAA6AA8A9v/x/+//8v/p/93/yv/G/9z/AgAtAGMAhwCKAFAACgDl/xMAWgBxAGwAOQDk/5//if+l//P/RwCUAKUATADj/5//gv+T/6L/z//j/93/DgBUAHIAagBNADIA+v+q/7z/y//f/+j/w/+w/7X/8v9HAG0APQDd/4j/ff+9//z/SABfABsA4v/Q/6r/zf8KADUAWwAfAL7/ef+B/7z/5v8kADYANABDAFAAQwAPANj/e/9z/67/wP/S/xkAJQDY/6P/nf+q/77/FwA6AFoALQC//67/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"
)
