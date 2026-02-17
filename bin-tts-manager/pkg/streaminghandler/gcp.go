package streaminghandler

import (
	"context"
	"fmt"
	"net"
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

	Stream  texttospeechpb.TextToSpeech_StreamingSynthesizeClient
	ConnAst net.Conn

	Message *message.Message

	muStream sync.Mutex // protects Stream
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

	stream, err := h.connect(ctx, voiceID, langCode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize GCP StreamingSynthesize")
	}

	cfCtx, cancel := context.WithCancel(context.Background())
	res := &GCPConfig{
		Streaming: st,
		Ctx:       cfCtx,
		Cancel:    cancel,
		Stream:    stream,
		ConnAst:   st.ConnAst,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID:         st.MessageID,
				CustomerID: st.CustomerID,
			},
			StreamingID: st.ID,
		},
		muStream: sync.Mutex{},
	}

	h.notifyHandler.PublishEvent(cfCtx, message.EventTypeInitiated, res.Message)

	return res, nil
}

func (h *gcpHandler) connect(ctx context.Context, voiceID string, langCode string) (texttospeechpb.TextToSpeech_StreamingSynthesizeClient, error) {
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
		return nil, errors.Wrapf(err, "could not create GCP TTS client")
	}

	stream, err := client.StreamingSynthesize(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start StreamingSynthesize")
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
					AudioEncoding:   texttospeechpb.AudioEncoding_LINEAR16,
					SampleRateHertz: defaultGCPStreamingSampleRate,
				},
			},
		},
	}
	if err := stream.Send(configReq); err != nil {
		return nil, errors.Wrapf(err, "could not send streaming config")
	}

	return stream, nil
}

func (h *gcpHandler) terminate(cf *GCPConfig) {
	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	if cf.Stream != nil {
		_ = cf.Stream.CloseSend()
		cf.Stream = nil
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
		cf.Cancel()
		h.notifyHandler.PublishEvent(cf.Ctx, message.EventTypePlayFinished, msg)
	}()

	for {
		select {
		case <-cf.Ctx.Done():
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
			log.Infof("GCP stream ended: %v", err)
			return
		}

		audioData := resp.GetAudioContent()
		if len(audioData) == 0 {
			continue
		}

		if errWrite := audiosocketWrite(cf.Ctx, cf.ConnAst, audioData); errWrite != nil {
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

	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	if cf.Stream == nil {
		return fmt.Errorf("GCP stream is nil")
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
		return errors.Wrapf(err, "failed to send text to GCP stream")
	}

	cf.Message.TotalMessage += text
	cf.Message.TotalCount++

	return nil
}

func (h *gcpHandler) SayFlush(vendorConfig any) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	// GCP gRPC doesn't have a flush concept like ElevenLabs.
	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	if cf.Stream == nil {
		return nil
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
