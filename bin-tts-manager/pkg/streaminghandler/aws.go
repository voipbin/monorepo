package streaminghandler

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// AWSConfig holds the state for a single AWS Polly streaming TTS session.
type AWSConfig struct {
	Streaming *streaming.Streaming

	Ctx    context.Context
	Cancel context.CancelFunc

	Client      *polly.Client
	ConnAst     *websocket.Conn
	ConnAstDone chan struct{} // closed when Asterisk WebSocket disconnects
	VoiceID     string

	Message *message.Message

	// audioCh receives PCM audio chunks from SayAdd calls
	audioCh chan []byte
	mu      sync.Mutex
}

const (
	defaultAWSStreamingRegion     = "eu-central-1"
	defaultAWSStreamingSampleRate = "8000"
	defaultAWSDefaultVoiceID      = "Joanna"
	defaultAWSAudioChBuffer       = 64
)

var awsVoiceIDMap = map[string]types.VoiceId{
	// English
	"english_male":    types.VoiceIdMatthew,
	"english_female":  types.VoiceIdJoanna,
	"english_neutral": types.VoiceIdJoanna,

	// Japanese
	"japanese_male":    types.VoiceIdTakumi,
	"japanese_female":  types.VoiceIdMizuki,
	"japanese_neutral": types.VoiceIdMizuki,

	// Chinese
	"chinese_male":    types.VoiceIdZhiyu,
	"chinese_female":  types.VoiceIdZhiyu,
	"chinese_neutral": types.VoiceIdZhiyu,

	// German
	"german_male":    types.VoiceIdHans,
	"german_female":  types.VoiceIdMarlene,
	"german_neutral": types.VoiceIdMarlene,

	// French
	"french_male":    types.VoiceIdMathieu,
	"french_female":  types.VoiceIdCeline,
	"french_neutral": types.VoiceIdCeline,

	// Korean
	"korean_male":    types.VoiceIdSeoyeon,
	"korean_female":  types.VoiceIdSeoyeon,
	"korean_neutral": types.VoiceIdSeoyeon,

	// Spanish
	"spanish_male":    types.VoiceIdEnrique,
	"spanish_female":  types.VoiceIdConchita,
	"spanish_neutral": types.VoiceIdConchita,

	// Portuguese
	"portuguese_male":    types.VoiceIdRicardo,
	"portuguese_female":  types.VoiceIdCamila,
	"portuguese_neutral": types.VoiceIdCamila,

	// Italian
	"italian_male":    types.VoiceIdGiorgio,
	"italian_female":  types.VoiceIdCarla,
	"italian_neutral": types.VoiceIdCarla,

	// Dutch
	"dutch_male":    types.VoiceIdRuben,
	"dutch_female":  types.VoiceIdLotte,
	"dutch_neutral": types.VoiceIdLotte,

	// Russian
	"russian_male":    types.VoiceIdMaxim,
	"russian_female":  types.VoiceIdTatyana,
	"russian_neutral": types.VoiceIdTatyana,

	// Arabic
	"arabic_male":    types.VoiceIdZeina,
	"arabic_female":  types.VoiceIdZeina,
	"arabic_neutral": types.VoiceIdZeina,

	// Hindi
	"hindi_male":    types.VoiceIdAditi,
	"hindi_female":  types.VoiceIdAditi,
	"hindi_neutral": types.VoiceIdAditi,

	// Polish
	"polish_male":    types.VoiceIdJacek,
	"polish_female":  types.VoiceIdEwa,
	"polish_neutral": types.VoiceIdEwa,
}

type awsHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	accessKey string
	secretKey string
}

func NewAWSHandler(reqHandler requesthandler.RequestHandler, notifyHandler notifyhandler.NotifyHandler, accessKey string, secretKey string) streamer {
	return &awsHandler{
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		accessKey:     accessKey,
		secretKey:     secretKey,
	}
}

func (h *awsHandler) Init(ctx context.Context, st *streaming.Streaming) (any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "awsHandler.Init",
		"streaming_id": st.ID,
	})

	client, err := h.createClient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create AWS Polly client")
	}

	voiceID := h.getVoiceID(ctx, st)
	log.Debugf("Using AWS Polly voice: %s", voiceID)

	cfCtx, cancel := context.WithCancel(context.Background())
	res := &AWSConfig{
		Streaming: st,
		Ctx:       cfCtx,
		Cancel:    cancel,
		Client:      client,
		ConnAst:     st.ConnAst,
		ConnAstDone: st.ConnAstDone,
		VoiceID:     voiceID,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID:         st.MessageID,
				CustomerID: st.CustomerID,
			},
			StreamingID: st.ID,
		},
		audioCh: make(chan []byte, defaultAWSAudioChBuffer),
		mu:      sync.Mutex{},
	}

	h.notifyHandler.PublishEvent(cfCtx, message.EventTypeInitiated, res.Message)

	return res, nil
}

func (h *awsHandler) createClient() (*polly.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(defaultAWSStreamingRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			h.accessKey,
			h.secretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %v", err)
	}

	return polly.NewFromConfig(cfg), nil
}

func (h *awsHandler) Run(vendorConfig any) error {
	cf, ok := vendorConfig.(*AWSConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *AWSConfig or is nil")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "awsHandler.Run",
		"streaming_id": cf.Streaming.ID,
	})

	go h.runProcess(cf)

	select {
	case <-cf.Ctx.Done():
	case <-cf.ConnAstDone:
		log.Infof("Asterisk WebSocket disconnected, tearing down AWS session")
	}
	cf.Cancel()

	return nil
}

func (h *awsHandler) runProcess(cf *AWSConfig) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "awsHandler.runProcess",
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

		case audioData, ok := <-cf.audioCh:
			if !ok {
				return
			}

			if len(audioData) == 0 {
				continue
			}

			if errWrite := websocketWrite(cf.Ctx, cf.ConnAst, audioData, frameSizeSlin); errWrite != nil {
				log.Errorf("Could not write audio to asterisk: %v", errWrite)
				return
			}
		}
	}
}

func (h *awsHandler) SayAdd(vendorConfig any, text string) error {
	log := logrus.WithField("func", "awsHandler.SayAdd")

	cf, ok := vendorConfig.(*AWSConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *AWSConfig or is nil")
	}

	input := &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		TextType:     types.TextTypeText,
		OutputFormat: types.OutputFormatPcm,
		VoiceId:      types.VoiceId(cf.VoiceID),
		SampleRate:   aws.String(defaultAWSStreamingSampleRate),
	}

	resp, err := cf.Client.SynthesizeSpeech(cf.Ctx, input)
	if err != nil {
		return errors.Wrapf(err, "failed to synthesize speech via AWS Polly")
	}
	defer func() {
		_ = resp.AudioStream.Close()
	}()

	// Read the audio stream synchronously and send to audioCh
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.AudioStream.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])

			select {
			case cf.audioCh <- chunk:
			case <-cf.Ctx.Done():
				return cf.Ctx.Err()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Errorf("Error reading AWS Polly audio stream: %v", readErr)
				return errors.Wrapf(readErr, "error reading AWS Polly audio stream")
			}
			break
		}
	}

	cf.Message.TotalMessage += text
	cf.Message.TotalCount++

	return nil
}

func (h *awsHandler) SayFlush(vendorConfig any) error {
	cf, ok := vendorConfig.(*AWSConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *AWSConfig or is nil")
	}

	// Drain any pending audio from the channel
	for {
		select {
		case <-cf.audioCh:
			continue
		default:
			return nil
		}
	}
}

func (h *awsHandler) SayStop(vendorConfig any) error {
	cf, ok := vendorConfig.(*AWSConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *AWSConfig or is nil")
	}

	cf.Cancel()
	return nil
}

func (h *awsHandler) SayFinish(vendorConfig any) error {
	cf, ok := vendorConfig.(*AWSConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *AWSConfig or is nil")
	}

	cf.Message.Finish = true

	// Cancel context first so any in-flight SayAdd takes the ctx.Done branch
	// instead of sending on the closed channel (which would panic).
	cf.Cancel()

	cf.mu.Lock()
	close(cf.audioCh)
	cf.mu.Unlock()

	return nil
}

// getVoiceID returns the AWS voice ID following the 3-tier fallback.
func (h *awsHandler) getVoiceID(ctx context.Context, st *streaming.Streaming) string {
	if st.VoiceID != "" {
		return st.VoiceID
	}

	if st.ActiveflowID != uuid.Nil {
		variables, err := h.reqHandler.FlowV1VariableGet(ctx, st.ActiveflowID)
		if err == nil {
			if res, ok := variables.Variables[variableAWSVoiceID]; ok && res != "" {
				return res
			}
		}
	}

	if tmpID := h.getVoiceIDByLangGender(st.Language, st.Gender); tmpID != "" {
		return string(tmpID)
	}

	return defaultAWSDefaultVoiceID
}

func (h *awsHandler) getVoiceIDByLangGender(language string, gender streaming.Gender) types.VoiceId {
	baseLang := strings.ToLower(strings.SplitN(language, "_", 2)[0])
	baseLang = strings.ToLower(strings.SplitN(baseLang, "-", 2)[0])
	tmpGender := strings.ToLower(string(gender))
	key := fmt.Sprintf("%s_%s", baseLang, tmpGender)

	if res, ok := awsVoiceIDMap[key]; ok {
		return res
	}

	neutralKey := fmt.Sprintf("%s_neutral", baseLang)
	if res, ok := awsVoiceIDMap[neutralKey]; ok {
		return res
	}

	return ""
}
