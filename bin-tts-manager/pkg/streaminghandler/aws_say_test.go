package streaminghandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

// newLocalPollyClient builds a Polly client that talks to a local test
// endpoint with synthetic credentials and without retries.
func newLocalPollyClient(endpoint string) *polly.Client {
	return polly.New(polly.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       "test-region-1",
		Credentials:  credentials.NewStaticCredentialsProvider("test-access-key", "test-secret-key", ""),
		Retryer:      aws.NopRetryer{},
	})
}

func newTestAWSConfig(client *polly.Client) *AWSConfig {
	ctx, cancel := context.WithCancel(context.Background())
	return &AWSConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("8d3e2f00-0000-4000-8000-000000000001")},
		},
		Ctx:     ctx,
		Cancel:  cancel,
		Client:  client,
		VoiceID: "Matthew",
		Message: &message.Message{},
		audioCh: make(chan []byte, defaultAWSAudioChBuffer),
	}
}

func Test_awsHandler_invalidVendorConfig(t *testing.T) {
	h := &awsHandler{}

	var typedNil *AWSConfig
	for name, cfg := range map[string]any{"wrong type": &GCPConfig{}, "untyped nil": nil, "typed nil": typedNil} {
		t.Run(name, func(t *testing.T) {
			if err := h.Run(cfg); err == nil {
				t.Error("Run: expected an error")
			}
			if err := h.SayAdd(cfg, "text"); err == nil {
				t.Error("SayAdd: expected an error")
			}
			if err := h.SayFlush(cfg); err == nil {
				t.Error("SayFlush: expected an error")
			}
			if err := h.SayStop(cfg); err == nil {
				t.Error("SayStop: expected an error")
			}
			if err := h.SayFinish(cfg); err == nil {
				t.Error("SayFinish: expected an error")
			}
		})
	}
}

func Test_awsHandler_SayAdd_streamsPCMToTheAudioChannel(t *testing.T) {
	// larger than the 4096 byte read buffer, so more than one chunk is produced
	pcm := make([]byte, 5000)
	for i := range pcm {
		pcm[i] = byte(i % 251)
	}

	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(pcm)
	}))
	defer srv.Close()

	cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
	h := &awsHandler{}

	if err := h.SayAdd(cf, "Hello world"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/v1/speech" {
		t.Errorf("Wrong request path. expected: /v1/speech, got: %s", gotPath)
	}
	for key, want := range map[string]string{
		"Text":         "Hello world",
		"VoiceId":      "Matthew",
		"OutputFormat": "pcm",
		"SampleRate":   "8000",
		"TextType":     "text",
	} {
		if gotBody[key] != want {
			t.Errorf("Wrong request field %s. expected: %q, got: %v", key, want, gotBody[key])
		}
	}

	close(cf.audioCh)
	var received []byte
	chunks := 0
	for chunk := range cf.audioCh {
		received = append(received, chunk...)
		chunks++
	}
	if chunks < 2 {
		t.Errorf("expected the stream to be split into chunks, got: %d", chunks)
	}
	if string(received) != string(pcm) {
		t.Errorf("the audio channel did not receive the exact PCM payload. got %d bytes, expected %d", len(received), len(pcm))
	}

	if cf.Message.TotalMessage != "Hello world" || cf.Message.TotalCount != 1 {
		t.Errorf("Wrong message state. got: %q %d", cf.Message.TotalMessage, cf.Message.TotalCount)
	}
}

func Test_awsHandler_SayAdd_errors(t *testing.T) {
	h := &awsHandler{}

	t.Run("service error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
		if err := h.SayAdd(cf, "text"); err == nil {
			t.Fatal("expected an error")
		}
		if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" || len(cf.audioCh) != 0 {
			t.Errorf("a failed synthesis must not change the state. got: %+v, queued: %d", cf.Message, len(cf.audioCh))
		}
	})

	t.Run("audio stream breaks in the middle", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/pcm")
			_, _ = w.Write([]byte{1, 2, 3, 4})
			w.(http.Flusher).Flush()
			panic(http.ErrAbortHandler)
		}))
		defer srv.Close()

		cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
		if err := h.SayAdd(cf, "text"); err == nil {
			t.Fatal("expected a stream read error")
		}
		if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
			t.Errorf("a broken stream must not count the message. got: %+v", cf.Message)
		}
	})
}

func Test_awsHandler_SayAdd_stopsWhenCancelledWhileTheAudioQueueIsFull(t *testing.T) {
	// more audio than the queue can hold, so SayAdd blocks on the send
	pcm := make([]byte, 4096*(defaultAWSAudioChBuffer+4))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(pcm)
	}))
	defer srv.Close()

	cf := newTestAWSConfig(newLocalPollyClient(srv.URL))
	h := &awsHandler{}

	result := make(chan error, 1)
	go func() { result <- h.SayAdd(cf, "text") }()

	deadline := time.Now().Add(5 * time.Second)
	for len(cf.audioCh) < cap(cf.audioCh) {
		if time.Now().After(deadline) {
			t.Fatal("the audio queue never filled up")
		}
		runtime.Gosched()
	}
	cf.Cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SayAdd did not return after the session was cancelled")
	}
	if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
		t.Errorf("a cancelled SayAdd must not count the message. got: %+v", cf.Message)
	}
}

func Test_awsHandler_SayFlush_dropsPendingAudio(t *testing.T) {
	cf := newTestAWSConfig(nil)
	cf.audioCh <- []byte{1}
	cf.audioCh <- []byte{2}
	cf.audioCh <- []byte{3}
	h := &awsHandler{}

	if err := h.SayFlush(cf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cf.audioCh) != 0 {
		t.Errorf("pending audio must be dropped. remaining: %d", len(cf.audioCh))
	}
	select {
	case <-cf.Ctx.Done():
		t.Error("a flush must not cancel the session")
	default:
	}
}

func Test_awsHandler_SayStop(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	if err := h.SayStop(cf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled")
	}
	if cf.Message.Finish {
		t.Error("stopping must not mark the message as finished")
	}
}

func Test_awsHandler_SayFinish(t *testing.T) {
	cf := newTestAWSConfig(nil)
	h := &awsHandler{}

	if err := h.SayFinish(cf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cf.Message.Finish {
		t.Error("Finish flag must be set")
	}
	select {
	case <-cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled")
	}
	if _, ok := <-cf.audioCh; ok {
		t.Error("the audio channel must be closed")
	}
}

func Test_awsHandler_getVoiceID(t *testing.T) {
	flowID := uuid.FromStringOrNil("8d3e2f00-0000-4000-8000-0000000000f1")

	tests := []struct {
		name string
		st   *streaming.Streaming

		expectLookup      bool
		responseVariables map[string]string
		variableErr       error

		expect string
	}{
		{
			name:   "explicit voice id wins without any lookup",
			st:     &streaming.Streaming{VoiceID: "Kendra", ActiveflowID: flowID, Language: "korean"},
			expect: "Kendra",
		},
		{
			name:              "activeflow variable wins over language",
			st:                &streaming.Streaming{ActiveflowID: flowID, Language: "korean", Gender: streaming.GenderMale},
			expectLookup:      true,
			responseVariables: map[string]string{variableAWSVoiceID: "Salli"},
			expect:            "Salli",
		},
		{
			name:              "empty variable falls through to language and gender",
			st:                &streaming.Streaming{ActiveflowID: flowID, Language: "english", Gender: streaming.GenderMale},
			expectLookup:      true,
			responseVariables: map[string]string{variableAWSVoiceID: ""},
			expect:            "Matthew",
		},
		{
			name:         "variable lookup error falls through to language and gender",
			st:           &streaming.Streaming{ActiveflowID: flowID, Language: "french", Gender: streaming.GenderFemale},
			expectLookup: true,
			variableErr:  errors.New("lookup failed"),
			expect:       "Celine",
		},
		{
			name:   "no activeflow uses language and gender without a lookup",
			st:     &streaming.Streaming{Language: "japanese", Gender: streaming.GenderMale},
			expect: "Takumi",
		},
		{
			name:   "unknown language falls back to the default voice",
			st:     &streaming.Streaming{Language: "klingon", Gender: streaming.GenderMale},
			expect: defaultAWSDefaultVoiceID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &awsHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectLookup {
				var res *fmvariable.Variable
				if tt.variableErr == nil {
					res = &fmvariable.Variable{Variables: tt.responseVariables}
				}
				mockReq.EXPECT().FlowV1VariableGet(ctx, tt.st.ActiveflowID).Return(res, tt.variableErr)
			}

			if got := h.getVoiceID(ctx, tt.st); got != tt.expect {
				t.Errorf("Wrong voice. expected: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_awsHandler_getVoiceIDByLangGender(t *testing.T) {
	h := &awsHandler{}

	tests := []struct {
		language string
		gender   streaming.Gender
		expect   types.VoiceId
	}{
		{"english", streaming.GenderMale, types.VoiceIdMatthew},
		{"English", streaming.GenderFemale, types.VoiceIdJoanna},
		{"english_us", streaming.GenderMale, types.VoiceIdMatthew},
		{"english-gb", streaming.GenderFemale, types.VoiceIdJoanna},
		{"german", streaming.GenderNeutral, types.VoiceIdMarlene},
		{"german", streaming.Gender("other"), types.VoiceIdMarlene},
		{"klingon", streaming.GenderMale, ""},
		{"", streaming.GenderFemale, ""},
	}

	for _, tt := range tests {
		t.Run(tt.language+"/"+string(tt.gender), func(t *testing.T) {
			if got := h.getVoiceIDByLangGender(tt.language, tt.gender); got != tt.expect {
				t.Errorf("Wrong voice. expected: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_awsHandler_runProcess(t *testing.T) {
	tests := []struct {
		name string
		// closeAst closes the Asterisk connection before audio arrives, to force a write failure.
		closeAst bool
		// chunks are queued to the audio channel in order.
		chunks [][]byte
		// closeChannel ends the loop the way SayFinish does.
		closeChannel bool

		expectFrames [][]byte
	}{
		{
			name:         "audio is forwarded and an empty chunk is skipped until the channel closes",
			chunks:       [][]byte{{1, 2, 3}, {}, {4, 5}},
			closeChannel: true,
			expectFrames: [][]byte{{1, 2, 3}, {4, 5}},
		},
		{
			name:     "a failed write to Asterisk ends the session",
			closeAst: true,
			chunks:   [][]byte{{9, 9}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			astClient, astServer := newLocalWSPair(t)
			cf := newTestAWSConfig(nil)
			cf.ConnAst = astClient
			t.Cleanup(cf.Cancel)
			if tt.closeAst {
				_ = astClient.Close()
			}

			finished := make(chan struct{})
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypePlayStarted, gomock.Any())
			mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypePlayFinished, gomock.Any()).Do(
				func(ctx context.Context, eventType string, data any) { close(finished) },
			)
			h := &awsHandler{notifyHandler: mockNotify}

			for _, chunk := range tt.chunks {
				cf.audioCh <- chunk
			}
			if tt.closeChannel {
				close(cf.audioCh)
			}

			go h.runProcess(cf)

			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Fatal("runProcess did not finish in time")
			}

			select {
			case <-cf.Ctx.Done():
			default:
				t.Error("runProcess must cancel the session context when it ends")
			}

			for i, want := range tt.expectFrames {
				if err := astServer.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
					t.Fatalf("could not set the read deadline: %v", err)
				}
				msgType, frame, err := astServer.ReadMessage()
				if err != nil {
					t.Fatalf("frame %d was not received: %v", i, err)
				}
				if msgType != websocket.BinaryMessage || string(frame) != string(want) {
					t.Errorf("Wrong frame %d. expected: %v, got type %d: %v", i, want, msgType, frame)
				}
			}
		})
	}
}
