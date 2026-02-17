# Multi-Provider Streaming TTS Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add GCP Cloud TTS (StreamingSynthesize) and AWS Polly (SynthesizeSpeech) as streaming TTS providers alongside ElevenLabs in bin-tts-manager.

**Architecture:** Each new provider implements the existing `streamer` interface (`Init`, `Run`, `SayStop`, `SayAdd`, `SayFlush`, `SayFinish`). The `runStreamer()` dispatch in `run.go` is changed to use `st.Provider` instead of iterating all handlers. GCP uses gRPC bidirectional streaming; AWS uses per-request HTTP with chunked response reading. Both output 8kHz PCM directly (no downsampling needed).

**Tech Stack:** Go, `cloud.google.com/go/texttospeech/apiv1` (already vendored), `github.com/aws/aws-sdk-go-v2/service/polly` (already vendored), AudioSocket protocol

**Design doc:** `docs/plans/2026-02-17-multi-provider-streaming-tts-design.md`

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts`

**All paths below are relative to `bin-tts-manager/` within the worktree.**

---

### Task 1: Add VendorName Constants and Flow Variable Constants

**Files:**
- Modify: `models/streaming/streaming.go:67-70`
- Modify: `pkg/streaminghandler/main.go:70-72`

**Step 1: Add GCP and AWS vendor name constants**

In `models/streaming/streaming.go`, add to the existing `VendorName` const block (after line 69):

```go
const (
	VendorNameNone       VendorName = ""           // vendor name is not set
	VendorNameElevenlabs VendorName = "elevenlabs" // elevenlabs vendor
	VendorNameGCP        VendorName = "gcp"        // gcp vendor
	VendorNameAWS        VendorName = "aws"        // aws vendor
)
```

**Step 2: Add flow variable constants**

In `pkg/streaminghandler/main.go`, add to the flow variable const block (after line 72):

```go
const (
	variableElevenlabsVoiceID = "voipbin.tts.elevenlabs.voice_id"
	variableGCPVoiceID        = "voipbin.tts.gcp.voice_id"
	variableAWSVoiceID        = "voipbin.tts.aws.voice_id"
)
```

**Step 3: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go build ./...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add models/streaming/streaming.go pkg/streaminghandler/main.go
git commit -m "NOJIRA-Add-gcp-aws-streaming-tts

- bin-tts-manager: Add VendorNameGCP and VendorNameAWS constants
- bin-tts-manager: Add flow variable constants for GCP and AWS voice IDs"
```

---

### Task 2: Update streamingHandler Struct and Constructor

**Files:**
- Modify: `pkg/streaminghandler/main.go:171-210`
- Modify: `cmd/tts-manager/main.go:130`

**Step 1: Add gcpHandler and awsHandler fields to streamingHandler struct**

In `pkg/streaminghandler/main.go`, replace the struct (lines 171-183):

```go
type streamingHandler struct {
	utilHandler    utilhandler.UtilHandler
	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler

	listenAddress string
	podID         string

	mapStreaming map[uuid.UUID]*streaming.Streaming
	muStreaming  sync.Mutex

	elevenlabsHandler streamer
	gcpHandler        streamer
	awsHandler        streamer
}
```

**Step 2: Update NewStreamingHandler to accept AWS credentials and create all handlers**

Replace the constructor (lines 185-210):

```go
// NewStreamingHandler define
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	listenAddress string,
	podID string,
	elevenlabsAPIKey string,
	awsAccessKey string,
	awsSecretKey string,
) StreamingHandler {

	elevenlabsHandler := NewElevenlabsHandler(reqHandler, notifyHandler, elevenlabsAPIKey)
	gcpHandler := NewGCPHandler(reqHandler, notifyHandler)
	awsHandler := NewAWSHandler(reqHandler, notifyHandler, awsAccessKey, awsSecretKey)

	return &streamingHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		requestHandler: reqHandler,
		notifyHandler:  notifyHandler,

		listenAddress: listenAddress,
		podID:         podID,

		mapStreaming: make(map[uuid.UUID]*streaming.Streaming),
		muStreaming:  sync.Mutex{},

		elevenlabsHandler: elevenlabsHandler,
		gcpHandler:        gcpHandler,
		awsHandler:        awsHandler,
	}
}
```

**Step 3: Update call site in cmd/tts-manager/main.go**

Replace line 130:

```go
streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, listenAddress, podID, config.Get().ElevenlabsAPIKey, config.Get().AWSAccessKey, config.Get().AWSSecretKey)
```

**Step 4: Create stub GCP and AWS handlers (needed to compile)**

These are minimal stubs that will be fully implemented in Tasks 4 and 5. Create them now so the code compiles.

Create `pkg/streaminghandler/gcp.go`:

```go
package streaminghandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"
)

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
	return nil, fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) Run(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayStop(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayAdd(vendorConfig any, text string) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayFlush(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}

func (h *gcpHandler) SayFinish(vendorConfig any) error {
	return fmt.Errorf("gcp streaming TTS not yet implemented")
}
```

Create `pkg/streaminghandler/aws.go`:

```go
package streaminghandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"
)

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
	return nil, fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) Run(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayStop(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayAdd(vendorConfig any, text string) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayFlush(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}

func (h *awsHandler) SayFinish(vendorConfig any) error {
	return fmt.Errorf("aws streaming TTS not yet implemented")
}
```

**Step 5: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go build ./...`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add pkg/streaminghandler/main.go pkg/streaminghandler/gcp.go pkg/streaminghandler/aws.go cmd/tts-manager/main.go
git commit -m "NOJIRA-Add-gcp-aws-streaming-tts

- bin-tts-manager: Add gcpHandler and awsHandler fields to streamingHandler
- bin-tts-manager: Update NewStreamingHandler to accept AWS credentials
- bin-tts-manager: Add stub GCP and AWS handler implementations"
```

---

### Task 3: Fix Provider Dispatch in runStreamer() and say.go

**Files:**
- Modify: `pkg/streaminghandler/run.go:132-181`
- Modify: `pkg/streaminghandler/say.go` (all switch statements)

**Step 1: Replace runStreamer() to dispatch based on st.Provider**

In `run.go`, replace the `runStreamer` function (lines 132-181):

```go
func (h *streamingHandler) runStreamer(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runStreamer",
		"streaming_id": st.ID,
		"provider":     st.Provider,
	})

	// Select handler based on provider
	handler, vendorName := h.getStreamerByProvider(st.Provider)
	if handler == nil {
		return fmt.Errorf("unsupported or unconfigured provider: %s", st.Provider)
	}

	tmp, errInit := handler.Init(ctx, st)
	if errInit != nil {
		log.Errorf("Handler initialization failed for provider %s: %v", st.Provider, errInit)
		return fmt.Errorf("could not initialize %s handler: %v", st.Provider, errInit)
	}

	h.SetVendorInfo(st, vendorName, tmp)

	go func(s *streaming.Streaming) {
		log.Debugf("Starting %s handler for streaming ID: %s", s.VendorName, s.ID)
		if errRun := handler.Run(s.VendorConfig); errRun != nil {
			log.Errorf("Could not run the %s handler. err: %v", s.VendorName, errRun)
		}
		log.Debugf("%s handler finished for streaming_id: %s, message_id: %s", s.VendorName, s.ID, s.MessageID)

		h.SetVendorInfo(s, streaming.VendorNameNone, nil)
	}(st)

	return nil
}

// getStreamerByProvider returns the streamer handler and vendor name for the given provider string.
func (h *streamingHandler) getStreamerByProvider(provider string) (streamer, streaming.VendorName) {
	switch streaming.VendorName(provider) {
	case streaming.VendorNameElevenlabs:
		return h.elevenlabsHandler, streaming.VendorNameElevenlabs
	case streaming.VendorNameGCP:
		return h.gcpHandler, streaming.VendorNameGCP
	case streaming.VendorNameAWS:
		return h.awsHandler, streaming.VendorNameAWS
	default:
		// Fallback: try elevenlabs for empty/unknown provider (backwards compat)
		return h.elevenlabsHandler, streaming.VendorNameElevenlabs
	}
}
```

**Step 2: Update SayStop in say.go to dispatch via getStreamerByProvider**

Replace `SayStop` (lines 40-61):

```go
func (h *streamingHandler) SayStop(ctx context.Context, id uuid.UUID) error {

	st, err := h.UpdateMessageID(ctx, id, uuid.Nil)
	if err != nil {
		return errors.Wrapf(err, "could not update message ID. streaming_id: %s, message_id: %s", id, uuid.Nil)
	}

	if st.VendorName == streaming.VendorNameNone || st.VendorConfig == nil {
		return nil
	}

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		return nil
	}

	return handler.SayStop(st.VendorConfig)
}
```

**Step 3: Update SayAdd in say.go to dispatch generically**

Replace `SayAdd` (lines 64-90):

```go
func (h *streamingHandler) SayAdd(ctx context.Context, id uuid.UUID, messageID uuid.UUID, text string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SayAdd",
		"streaming_id": id,
		"text":         text,
	})

	st, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get streaming info. streaming_id: %s, message_id: %s", id, messageID)
	}

	if st.MessageID != messageID {
		return fmt.Errorf("message ID mismatch. streaming_id: %s, current_message_id: %s, request_message_id: %s", id, st.MessageID, messageID)
	} else if st.VendorConfig == nil {
		return fmt.Errorf("vendor config is nil. streaming_id: %s", id)
	}

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		return errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}

	log.Debugf("Adding text to %s streaming. streaming_id: %s, message_id: %s, text: %s", st.VendorName, id, messageID, text)
	return handler.SayAdd(st.VendorConfig, text)
}
```

**Step 4: Update SayFlush in say.go**

Replace `SayFlush` (lines 96-124):

```go
func (h *streamingHandler) SayFlush(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "SayFlush",
		"streaming_id": id,
	})

	st, err := h.Get(ctx, id)
	if err != nil {
		log.Infof("Could not get streaming. err: %v", err)
		return err
	}

	st.VendorLock.Lock()
	defer st.VendorLock.Unlock()

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		log.Errorf("Unsupported vendor. vendor_name: %s", st.VendorName)
		return fmt.Errorf("unsupported vendor: %s", st.VendorName)
	}

	if errFlush := handler.SayFlush(st.VendorConfig); errFlush != nil {
		log.Errorf("Could not flush the say streaming. err: %v", errFlush)
		return errFlush
	}

	return nil
}
```

**Step 5: Update SayFinish in say.go**

Replace `SayFinish` (lines 127-149):

```go
func (h *streamingHandler) SayFinish(ctx context.Context, id uuid.UUID, messageID uuid.UUID) (*streaming.Streaming, error) {
	st, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get streaming info. streaming_id: %s, message_id: %s", id, messageID)
	}

	if st.MessageID != messageID {
		return nil, fmt.Errorf("message ID mismatch. streaming_id: %s, current_message_id: %s, request_message_id: %s", id, st.MessageID, messageID)
	} else if st.VendorConfig == nil {
		return nil, fmt.Errorf("vendor config is nil. streaming_id: %s", id)
	}

	handler, _ := h.getStreamerByProvider(string(st.VendorName))
	if handler == nil {
		return nil, errors.Errorf("unsupported vendor for text streaming. vendor: %s", st.VendorName)
	}

	if errFinish := handler.SayFinish(st.VendorConfig); errFinish != nil {
		return nil, errors.Wrapf(errFinish, "could not finish streaming. streaming_id: %s, message_id: %s", id, messageID)
	}
	return st, nil
}
```

**Step 6: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go build ./...`
Expected: SUCCESS

**Step 7: Commit**

```bash
git add pkg/streaminghandler/run.go pkg/streaminghandler/say.go
git commit -m "NOJIRA-Add-gcp-aws-streaming-tts

- bin-tts-manager: Replace hardcoded runStreamer dispatch with provider-based lookup
- bin-tts-manager: Update all Say* methods to use getStreamerByProvider"
```

---

### Task 4: Implement GCP Cloud TTS StreamingSynthesize Handler

**Files:**
- Modify: `pkg/streaminghandler/gcp.go` (replace stub)

**Step 1: Implement the full GCP handler**

Replace the entire contents of `pkg/streaminghandler/gcp.go`:

```go
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

	Stream texttospeechpb.TextToSpeech_StreamingSynthesizeClient
	ConnAst net.Conn

	Message *message.Message

	muStream sync.Mutex // protects Stream
}

const (
	defaultGCPStreamingEndpoint   = "eu-texttospeech.googleapis.com:443"
	defaultGCPStreamingSampleRate = int32(8000)
	defaultGCPDefaultVoiceID      = "en-US-Chirp3-HD-Charon"
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
	// Send an empty text to signal flush intent.
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
```

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go build ./...`
Expected: SUCCESS (may need `go mod tidy && go mod vendor` first if new imports are needed)

**Step 3: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/streaminghandler/gcp.go
git commit -m "NOJIRA-Add-gcp-aws-streaming-tts

- bin-tts-manager: Implement GCP Cloud TTS StreamingSynthesize handler
- bin-tts-manager: Add Chirp3-HD voice mapping for 14 languages"
```

---

### Task 5: Implement AWS Polly SynthesizeSpeech Handler

**Files:**
- Modify: `pkg/streaminghandler/aws.go` (replace stub)

**Step 1: Implement the full AWS handler**

Replace the entire contents of `pkg/streaminghandler/aws.go`:

```go
package streaminghandler

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// AWSConfig holds the state for a single AWS Polly streaming TTS session.
type AWSConfig struct {
	Streaming *streaming.Streaming

	Ctx    context.Context
	Cancel context.CancelFunc

	Client  *polly.Client
	ConnAst net.Conn

	Message *message.Message

	// audioCh receives PCM audio chunks from SayAdd goroutines
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
		Client:    client,
		ConnAst:   st.ConnAst,
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

	go h.runProcess(cf)

	<-cf.Ctx.Done()

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

			if errWrite := audiosocketWrite(cf.Ctx, cf.ConnAst, audioData); errWrite != nil {
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

	voiceID := h.getVoiceID(cf.Ctx, cf.Streaming)

	input := &polly.SynthesizeSpeechInput{
		Text:         aws.String(text),
		TextType:     types.TextTypeText,
		OutputFormat: types.OutputFormatPcm,
		VoiceId:      types.VoiceId(voiceID),
		SampleRate:   aws.String(defaultAWSStreamingSampleRate),
	}

	resp, err := cf.Client.SynthesizeSpeech(cf.Ctx, input)
	if err != nil {
		return errors.Wrapf(err, "failed to synthesize speech via AWS Polly")
	}

	// Read the audio stream in chunks and send to audioCh
	go func() {
		defer func() {
			_ = resp.AudioStream.Close()
		}()

		buf := make([]byte, 4096)
		for {
			n, readErr := resp.AudioStream.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])

				select {
				case cf.audioCh <- chunk:
				case <-cf.Ctx.Done():
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					log.Errorf("Error reading AWS Polly audio stream: %v", readErr)
				}
				return
			}
		}
	}()

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

// Ensure strconv is used (for defaultAWSStreamingSampleRate validation if needed)
var _ = strconv.Itoa
```

**Note:** Remove the `var _ = strconv.Itoa` line if the linter complains. It's there only if needed for validation later. If `strconv` is unused, remove both the import and the line.

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go mod tidy && go mod vendor && go build ./...`
Expected: SUCCESS

**Step 3: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/streaminghandler/aws.go
git commit -m "NOJIRA-Add-gcp-aws-streaming-tts

- bin-tts-manager: Implement AWS Polly SynthesizeSpeech streaming handler
- bin-tts-manager: Add Polly voice mapping for 14 languages"
```

---

### Task 6: Run Full Verification Workflow

**Step 1: Run the full verification**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts/bin-tts-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All steps pass.

**Step 2: Fix any lint issues**

Common issues to watch for:
- Unused imports (`strconv` in aws.go if not needed)
- `staticcheck` warnings on GCP client creation (use `//nolint:staticcheck` like the existing code)

**Step 3: Commit any fixes**

```bash
git add -A
git commit -m "NOJIRA-Add-gcp-aws-streaming-tts

- bin-tts-manager: Fix lint and verification issues"
```

---

### Task 7: Push Branch and Create PR

**Step 1: Fetch latest main and check for conflicts**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Add-gcp-aws-streaming-tts
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

Expected: No conflicts.

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Add-gcp-aws-streaming-tts
```

Create PR with title `NOJIRA-Add-gcp-aws-streaming-tts` and body describing all changes.
