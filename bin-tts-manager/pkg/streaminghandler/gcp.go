package streaminghandler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// GCPConfig holds the state for a single GCP streaming TTS session.
type GCPConfig struct {
	Streaming *streaming.Streaming

	Ctx    context.Context
	Cancel context.CancelFunc

	StreamCtx    context.Context    // per-stream sub-context, cancelled by SayFlush
	StreamCancel context.CancelFunc

	Client  *texttospeech.Client
	Stream  texttospeechpb.TextToSpeech_StreamingSynthesizeClient
	ConnAst *websocket.Conn

	VoiceID  string // stored for reconnect after flush
	LangCode string // stored for reconnect after flush

	Message *message.Message

	processDone chan struct{} // closed when runProcess exits
	muStream    sync.Mutex   // protects Stream, Client, StreamCtx/StreamCancel
}

const (
	defaultGCPStreamingEndpoint   = "eu-texttospeech.googleapis.com:443"
	defaultGCPStreamingSampleRate = int32(8000)
	defaultGCPDefaultVoiceID     = "en-US-Chirp3-HD-Charon"
)

var gcpVoiceIDMap = map[string]string{
	// English
	"english_male":    "en-US-Chirp3-HD-Charon",
	"english_female":  "en-US-Chirp3-HD-Aoede",
	"english_neutral": "en-US-Chirp3-HD-Aoede",

	// Japanese
	"japanese_male":    "ja-JP-Chirp3-HD-Charon",
	"japanese_female":  "ja-JP-Chirp3-HD-Aoede",
	"japanese_neutral": "ja-JP-Chirp3-HD-Aoede",

	// Chinese
	"chinese_male":    "cmn-CN-Chirp3-HD-Charon",
	"chinese_female":  "cmn-CN-Chirp3-HD-Aoede",
	"chinese_neutral": "cmn-CN-Chirp3-HD-Aoede",

	// German
	"german_male":    "de-DE-Chirp3-HD-Charon",
	"german_female":  "de-DE-Chirp3-HD-Aoede",
	"german_neutral": "de-DE-Chirp3-HD-Aoede",

	// French
	"french_male":    "fr-FR-Chirp3-HD-Charon",
	"french_female":  "fr-FR-Chirp3-HD-Aoede",
	"french_neutral": "fr-FR-Chirp3-HD-Aoede",

	// Korean
	"korean_male":    "ko-KR-Chirp3-HD-Charon",
	"korean_female":  "ko-KR-Chirp3-HD-Aoede",
	"korean_neutral": "ko-KR-Chirp3-HD-Aoede",

	// Spanish
	"spanish_male":    "es-ES-Chirp3-HD-Charon",
	"spanish_female":  "es-ES-Chirp3-HD-Aoede",
	"spanish_neutral": "es-ES-Chirp3-HD-Aoede",

	// Portuguese
	"portuguese_male":    "pt-BR-Chirp3-HD-Charon",
	"portuguese_female":  "pt-BR-Chirp3-HD-Aoede",
	"portuguese_neutral": "pt-BR-Chirp3-HD-Aoede",

	// Italian
	"italian_male":    "it-IT-Chirp3-HD-Charon",
	"italian_female":  "it-IT-Chirp3-HD-Aoede",
	"italian_neutral": "it-IT-Chirp3-HD-Aoede",

	// Dutch
	"dutch_male":    "nl-NL-Chirp3-HD-Charon",
	"dutch_female":  "nl-NL-Chirp3-HD-Aoede",
	"dutch_neutral": "nl-NL-Chirp3-HD-Aoede",

	// Russian
	"russian_male":    "ru-RU-Chirp3-HD-Charon",
	"russian_female":  "ru-RU-Chirp3-HD-Aoede",
	"russian_neutral": "ru-RU-Chirp3-HD-Aoede",

	// Arabic
	"arabic_male":    "ar-XA-Chirp3-HD-Charon",
	"arabic_female":  "ar-XA-Chirp3-HD-Aoede",
	"arabic_neutral": "ar-XA-Chirp3-HD-Aoede",

	// Hindi
	"hindi_male":    "hi-IN-Chirp3-HD-Charon",
	"hindi_female":  "hi-IN-Chirp3-HD-Aoede",
	"hindi_neutral": "hi-IN-Chirp3-HD-Aoede",

	// Polish
	"polish_male":    "pl-PL-Chirp3-HD-Charon",
	"polish_female":  "pl-PL-Chirp3-HD-Aoede",
	"polish_neutral": "pl-PL-Chirp3-HD-Aoede",
}

type gcpHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

func NewGCPHandler(reqHandler requesthandler.RequestHandler, notifyHandler notifyhandler.NotifyHandler) streamer {
	return &gcpHandler{
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}
}

func (h *gcpHandler) Init(ctx context.Context, st *streaming.Streaming) (any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "gcpHandler.Init",
		"streaming_id": st.ID,
	})

	voiceID := h.getVoiceID(ctx, st)
	log.Debugf("Using GCP voice: %s", voiceID)

	// Extract language code from the voice name (e.g., "en-US-Chirp3-HD-Charon" -> "en-US")
	langCode := h.extractLangCode(voiceID, st.Language)

	client, stream, err := h.connect(ctx, voiceID, langCode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize GCP StreamingSynthesize")
	}

	cfCtx, cfCancel := context.WithCancel(context.Background())
	streamCtx, streamCancel := context.WithCancel(cfCtx)

	res := &GCPConfig{
		Streaming:    st,
		Ctx:          cfCtx,
		Cancel:       cfCancel,
		StreamCtx:    streamCtx,
		StreamCancel: streamCancel,
		Client:       client,
		Stream:       stream,
		ConnAst:      st.ConnAst,
		VoiceID:      voiceID,
		LangCode:     langCode,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID:         st.MessageID,
				CustomerID: st.CustomerID,
			},
			StreamingID: st.ID,
		},
		processDone: make(chan struct{}),
		muStream:    sync.Mutex{},
	}

	h.notifyHandler.PublishEvent(cfCtx, message.EventTypeInitiated, res.Message)

	return res, nil
}

func (h *gcpHandler) connect(ctx context.Context, voiceID string, langCode string) (*texttospeech.Client, texttospeechpb.TextToSpeech_StreamingSynthesizeClient, error) {
	keepAliveParams := keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: true,
	}

	//nolint:staticcheck
	client, err := texttospeech.NewClient(
		ctx,
		option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepAliveParams)),
		option.WithEndpoint(defaultGCPStreamingEndpoint),
	)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not create GCP TTS client")
	}

	stream, err := client.StreamingSynthesize(ctx)
	if err != nil {
		_ = client.Close()
		return nil, nil, errors.Wrapf(err, "could not start StreamingSynthesize")
	}

	// Send config as the first message
	configReq := &texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: &texttospeechpb.StreamingSynthesizeConfig{
				Voice: &texttospeechpb.VoiceSelectionParams{
					LanguageCode: langCode,
					Name:         voiceID,
				},
				StreamingAudioConfig: &texttospeechpb.StreamingAudioConfig{
					AudioEncoding:   texttospeechpb.AudioEncoding_MULAW,
					SampleRateHertz: defaultGCPStreamingSampleRate,
				},
			},
		},
	}
	if err := stream.Send(configReq); err != nil {
		_ = client.Close()
		return nil, nil, errors.Wrapf(err, "could not send streaming config")
	}

	return client, stream, nil
}

func (h *gcpHandler) terminate(cf *GCPConfig) {
	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	if cf.Stream != nil {
		_ = cf.Stream.CloseSend()
		cf.Stream = nil
	}

	if cf.Client != nil {
		_ = cf.Client.Close()
		cf.Client = nil
	}

	cf.Cancel()
}

func (h *gcpHandler) Run(vendorConfig any) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	go h.runProcess(cf)

	<-cf.Ctx.Done()
	h.terminate(cf)

	return nil
}

func (h *gcpHandler) runProcess(cf *GCPConfig) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "gcpHandler.runProcess",
		"streaming_id": cf.Streaming.ID,
	})

	msg := cf.Message
	h.notifyHandler.PublishEvent(cf.Ctx, message.EventTypePlayStarted, msg)

	defer func() {
		cf.StreamCancel()
		h.notifyHandler.PublishEvent(cf.Ctx, message.EventTypePlayFinished, msg)
		close(cf.processDone)
	}()

	for {
		select {
		case <-cf.StreamCtx.Done():
			return
		default:
		}

		cf.muStream.Lock()
		stream := cf.Stream
		cf.muStream.Unlock()

		if stream == nil {
			return
		}

		resp, err := stream.Recv()
		if err != nil {
			// If the per-stream context was cancelled (SayFlush or SayStop),
			// the stream/client are already nil — just return and let the defer run.
			select {
			case <-cf.StreamCtx.Done():
				log.Infof("GCP stream ended (context cancelled): %v", err)
				return
			default:
			}

			// Stream died server-side (e.g., GCP 5s inactivity timeout).
			// Nil out the stream so the next SayAdd can reconnect.
			log.Infof("GCP stream ended, will reconnect on next SayAdd: %v", err)
			cf.muStream.Lock()
			if cf.Stream != nil {
				_ = cf.Stream.CloseSend()
				cf.Stream = nil
			}
			if cf.Client != nil {
				_ = cf.Client.Close()
				cf.Client = nil
			}
			cf.muStream.Unlock()
			return
		}

		audioData := resp.GetAudioContent()
		if len(audioData) == 0 {
			continue
		}

		if errWrite := websocketWrite(cf.StreamCtx, cf.ConnAst, audioData); errWrite != nil {
			log.Errorf("Could not write audio to asterisk: %v", errWrite)
			return
		}
	}
}

func (h *gcpHandler) SayAdd(vendorConfig any, text string) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "gcpHandler.SayAdd",
		"streaming_id": cf.Streaming.ID,
	})

	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	// Reconnect if stream is nil (died from GCP timeout, flushed, etc.)
	if cf.Stream == nil {
		log.Infof("GCP stream is nil, reconnecting")
		if err := h.waitAndReconnectLocked(cf); err != nil {
			return err
		}
	}

	req := &texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_Input{
			Input: &texttospeechpb.StreamingSynthesisInput{
				InputSource: &texttospeechpb.StreamingSynthesisInput_Text{
					Text: text,
				},
			},
		},
	}

	if err := cf.Stream.Send(req); err != nil {
		// Send failed — stream may have died between our nil check and the Send.
		// Clean up, wait for runProcess to exit, and reconnect once.
		log.Infof("Send failed, reconnecting: %v", err)
		if cf.Stream != nil {
			_ = cf.Stream.CloseSend()
			cf.Stream = nil
		}
		if cf.Client != nil {
			_ = cf.Client.Close()
			cf.Client = nil
		}

		if errReconnect := h.waitAndReconnectLocked(cf); errReconnect != nil {
			return errors.Wrapf(err, "failed to send text and reconnect GCP stream")
		}

		if errRetry := cf.Stream.Send(req); errRetry != nil {
			return errors.Wrapf(errRetry, "failed to send text after GCP stream reconnect")
		}
	}

	cf.Message.TotalMessage += text
	cf.Message.TotalCount++

	return nil
}

// waitAndReconnectLocked waits for the previous runProcess goroutine to exit,
// then creates a new GCP client/stream and launches a new runProcess.
// Must be called with cf.muStream held. Temporarily releases the lock while
// waiting on processDone to avoid deadlock with runProcess.
func (h *gcpHandler) waitAndReconnectLocked(cf *GCPConfig) error {
	// Release lock while waiting — runProcess needs it to read cf.Stream.
	cf.muStream.Unlock()
	<-cf.processDone
	cf.muStream.Lock()

	// Re-check: another goroutine may have already reconnected.
	if cf.Stream != nil {
		return nil
	}

	client, stream, err := h.connect(cf.Ctx, cf.VoiceID, cf.LangCode)
	if err != nil {
		return errors.Wrapf(err, "failed to reconnect GCP stream")
	}
	cf.Client = client
	cf.Stream = stream

	streamCtx, streamCancel := context.WithCancel(cf.Ctx)
	cf.StreamCtx = streamCtx
	cf.StreamCancel = streamCancel

	cf.processDone = make(chan struct{})
	go h.runProcess(cf)

	return nil
}

func (h *gcpHandler) SayFlush(vendorConfig any) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	cf.StreamCancel()

	if cf.Stream != nil {
		_ = cf.Stream.CloseSend()
		cf.Stream = nil
	}

	if cf.Client != nil {
		_ = cf.Client.Close()
		cf.Client = nil
	}

	return nil
}

func (h *gcpHandler) SayStop(vendorConfig any) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	cf.Cancel()
	return nil
}

func (h *gcpHandler) SayFinish(vendorConfig any) error {
	log := logrus.WithField("func", "gcpHandler.SayFinish")

	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	cf.Message.Finish = true

	// Close the send direction to signal we're done sending text
	cf.muStream.Lock()
	if cf.Stream != nil {
		_ = cf.Stream.CloseSend()
	}
	cf.muStream.Unlock()

	log.Debugf("SayFinish called. Waiting for remaining audio.")

	return nil
}

// getVoiceID returns the GCP voice ID following the 3-tier fallback.
func (h *gcpHandler) getVoiceID(ctx context.Context, st *streaming.Streaming) string {
	if st.VoiceID != "" {
		return st.VoiceID
	}

	if st.ActiveflowID != uuid.Nil {
		variables, err := h.reqHandler.FlowV1VariableGet(ctx, st.ActiveflowID)
		if err == nil {
			if res, ok := variables.Variables[variableGCPVoiceID]; ok && res != "" {
				return res
			}
		}
	}

	if tmpID := h.getVoiceIDByLangGender(st.Language, st.Gender); tmpID != "" {
		return tmpID
	}

	return defaultGCPDefaultVoiceID
}

func (h *gcpHandler) getVoiceIDByLangGender(language string, gender streaming.Gender) string {
	baseLang := strings.ToLower(strings.SplitN(language, "_", 2)[0])
	baseLang = strings.ToLower(strings.SplitN(baseLang, "-", 2)[0])
	tmpGender := strings.ToLower(string(gender))
	key := fmt.Sprintf("%s_%s", baseLang, tmpGender)

	if res, ok := gcpVoiceIDMap[key]; ok {
		return res
	}

	neutralKey := fmt.Sprintf("%s_neutral", baseLang)
	if res, ok := gcpVoiceIDMap[neutralKey]; ok {
		return res
	}

	return ""
}

// extractLangCode extracts the language code from a Chirp3 voice name.
// e.g., "en-US-Chirp3-HD-Charon" -> "en-US"
func (h *gcpHandler) extractLangCode(voiceID string, fallbackLang string) string {
	parts := strings.SplitN(voiceID, "-Chirp3", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	if fallbackLang != "" {
		return fallbackLang
	}

	return "en-US"
}
