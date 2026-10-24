package streaminghandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// fakeGCPStream implements only the calls the handler makes on the GCP
// stream; every other method of the embedded interface would panic.
type fakeGCPStream struct {
	texttospeechpb.TextToSpeech_StreamingSynthesizeClient

	sendErr        error
	sent           []*texttospeechpb.StreamingSynthesizeRequest
	closeSendCalls int
}

func (f *fakeGCPStream) Send(req *texttospeechpb.StreamingSynthesizeRequest) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, req)
	return nil
}

func (f *fakeGCPStream) CloseSend() error {
	f.closeSendCalls++
	return nil
}

func newTestGCPConfig(stream *fakeGCPStream) *GCPConfig {
	ctx, cancel := context.WithCancel(context.Background())
	streamCtx, streamCancel := context.WithCancel(ctx)

	cf := &GCPConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("6c2d1e00-0000-4000-8000-000000000001")},
		},
		Ctx:          ctx,
		Cancel:       cancel,
		StreamCtx:    streamCtx,
		StreamCancel: streamCancel,
		Message:      &message.Message{},
		processDone:  make(chan struct{}),
	}
	if stream != nil {
		cf.Stream = stream
	}

	return cf
}

func Test_gcpHandler_invalidVendorConfig(t *testing.T) {
	h := &gcpHandler{}

	var typedNil *GCPConfig
	for name, cfg := range map[string]any{"wrong type": &AWSConfig{}, "untyped nil": nil, "typed nil": typedNil} {
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

func Test_gcpHandler_SayAdd_sendsText(t *testing.T) {
	stream := &fakeGCPStream{}
	cf := newTestGCPConfig(stream)
	h := &gcpHandler{}

	before := time.Now()
	if err := h.SayAdd(cf, "Hello "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.SayAdd(cf, "world"); err != nil {
		t.Fatalf("unexpected error on the second call: %v", err)
	}

	if len(stream.sent) != 2 {
		t.Fatalf("expected 2 requests sent, got: %d", len(stream.sent))
	}
	if got := stream.sent[0].GetInput().GetText(); got != "Hello " {
		t.Errorf("Wrong first text. expected: %q, got: %q", "Hello ", got)
	}
	if got := stream.sent[1].GetInput().GetText(); got != "world" {
		t.Errorf("Wrong second text. expected: %q, got: %q", "world", got)
	}
	if cf.Message.TotalMessage != "Hello world" || cf.Message.TotalCount != 2 {
		t.Errorf("Wrong message state. got: %q %d", cf.Message.TotalMessage, cf.Message.TotalCount)
	}
	if cf.lastSendTime.Before(before) {
		t.Error("lastSendTime must be refreshed by SayAdd")
	}
}

func Test_gcpHandler_SayAdd_reconnectRefusedWhenSessionIsCancelled(t *testing.T) {
	tests := []struct {
		name   string
		stream *fakeGCPStream
		// expectCloseSend is how many times CloseSend must have been called on the stream.
		expectCloseSend int
		expectSent      int
	}{
		{
			name:            "stream is gone",
			stream:          nil,
			expectCloseSend: 0,
		},
		{
			name:            "send fails, so the stream is closed before the reconnect is refused",
			stream:          &fakeGCPStream{sendErr: errors.New("stream broken")},
			expectCloseSend: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := newTestGCPConfig(tt.stream)
			// the previous runProcess has not finished (processDone stays open)
			// and the session is cancelled, so the reconnect must not dial.
			cf.Cancel()
			h := &gcpHandler{}

			err := h.SayAdd(cf, "text")
			if err == nil || !strings.Contains(err.Error(), "session context cancelled") {
				t.Fatalf("expected a session cancelled error, got: %v", err)
			}
			if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
				t.Errorf("a failed SayAdd must not count the message. got: %+v", cf.Message)
			}
			if cf.Stream != nil {
				t.Error("no stream may remain after the failure")
			}
			if tt.stream != nil && tt.stream.closeSendCalls != tt.expectCloseSend {
				t.Errorf("Wrong CloseSend count. expected: %d, got: %d", tt.expectCloseSend, tt.stream.closeSendCalls)
			}
		})
	}
}

func Test_gcpHandler_SayFlush(t *testing.T) {
	stream := &fakeGCPStream{}
	cf := newTestGCPConfig(stream)
	h := &gcpHandler{}

	if err := h.SayFlush(cf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-cf.StreamCtx.Done():
	default:
		t.Error("the stream context must be cancelled")
	}
	select {
	case <-cf.Ctx.Done():
		t.Error("a flush must not cancel the whole session")
	default:
	}
	if stream.closeSendCalls != 1 || cf.Stream != nil {
		t.Errorf("the stream must be closed once and released. close calls: %d, stream: %v", stream.closeSendCalls, cf.Stream)
	}
}

func Test_gcpHandler_SayStop(t *testing.T) {
	cf := newTestGCPConfig(nil)
	h := &gcpHandler{}

	if err := h.SayStop(cf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled")
	}
}

func Test_gcpHandler_SayFinish(t *testing.T) {
	h := &gcpHandler{}

	t.Run("closes the send direction", func(t *testing.T) {
		stream := &fakeGCPStream{}
		cf := newTestGCPConfig(stream)

		if err := h.SayFinish(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cf.Message.Finish || stream.closeSendCalls != 1 {
			t.Errorf("expected Finish and one CloseSend. got: %v, %d", cf.Message.Finish, stream.closeSendCalls)
		}
		if cf.Stream == nil {
			t.Error("the stream must stay so that the remaining audio can be received")
		}
	})

	t.Run("no stream", func(t *testing.T) {
		cf := newTestGCPConfig(nil)
		if err := h.SayFinish(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cf.Message.Finish {
			t.Error("Finish flag must be set")
		}
	})
}

func Test_gcpHandler_terminate(t *testing.T) {
	stream := &fakeGCPStream{}
	cf := newTestGCPConfig(stream)
	h := &gcpHandler{}

	h.terminate(cf)

	if stream.closeSendCalls != 1 || cf.Stream != nil || cf.Client != nil {
		t.Errorf("the stream must be closed and released. close calls: %d", stream.closeSendCalls)
	}
	select {
	case <-cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled")
	}
}

func Test_gcpHandler_getVoiceID(t *testing.T) {
	flowID := uuid.FromStringOrNil("6c2d1e00-0000-4000-8000-0000000000f1")

	tests := []struct {
		name string
		st   *streaming.Streaming

		// responseVariables is returned for the activeflow; nil with variableErr unset means "not called".
		expectLookup      bool
		responseVariables map[string]string
		variableErr       error

		expect string
	}{
		{
			name:   "explicit voice id wins without any lookup",
			st:     &streaming.Streaming{VoiceID: "custom-voice", ActiveflowID: flowID, Language: "korean"},
			expect: "custom-voice",
		},
		{
			name:              "activeflow variable wins over language",
			st:                &streaming.Streaming{ActiveflowID: flowID, Language: "korean", Gender: streaming.GenderMale},
			expectLookup:      true,
			responseVariables: map[string]string{variableGCPVoiceID: "ja-JP-Chirp3-HD-Charon"},
			expect:            "ja-JP-Chirp3-HD-Charon",
		},
		{
			name:              "empty variable falls through to language and gender",
			st:                &streaming.Streaming{ActiveflowID: flowID, Language: "korean", Gender: streaming.GenderMale},
			expectLookup:      true,
			responseVariables: map[string]string{variableGCPVoiceID: ""},
			expect:            "ko-KR-Chirp3-HD-Charon",
		},
		{
			name:         "variable lookup error falls through to language and gender",
			st:           &streaming.Streaming{ActiveflowID: flowID, Language: "french", Gender: streaming.GenderFemale},
			expectLookup: true,
			variableErr:  errors.New("lookup failed"),
			expect:       "fr-FR-Chirp3-HD-Aoede",
		},
		{
			name:   "no activeflow uses language and gender without a lookup",
			st:     &streaming.Streaming{Language: "english", Gender: streaming.GenderMale},
			expect: "en-US-Chirp3-HD-Charon",
		},
		{
			name:   "unknown language falls back to the default voice",
			st:     &streaming.Streaming{Language: "klingon", Gender: streaming.GenderMale},
			expect: defaultGCPDefaultVoiceID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &gcpHandler{reqHandler: mockReq}
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

func Test_gcpHandler_getVoiceIDByLangGender(t *testing.T) {
	h := &gcpHandler{}

	tests := []struct {
		language string
		gender   streaming.Gender
		expect   string
	}{
		{"english", streaming.GenderMale, "en-US-Chirp3-HD-Charon"},
		{"English", streaming.GenderFemale, "en-US-Chirp3-HD-Aoede"},
		{"english_us", streaming.GenderMale, "en-US-Chirp3-HD-Charon"},
		{"english-gb", streaming.GenderFemale, "en-US-Chirp3-HD-Aoede"},
		{"korean", streaming.GenderNeutral, "ko-KR-Chirp3-HD-Aoede"},
		{"korean", streaming.Gender("other"), "ko-KR-Chirp3-HD-Aoede"},
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

func Test_gcpHandler_extractLangCode(t *testing.T) {
	h := &gcpHandler{}

	tests := []struct {
		name     string
		voiceID  string
		fallback string
		expect   string
	}{
		{"chirp voice", "ko-KR-Chirp3-HD-Charon", "xx-XX", "ko-KR"},
		{"voice without the chirp suffix is taken as is", "custom-voice", "xx-XX", "custom-voice"},
		{"voice starting with the suffix uses the fallback", "-Chirp3-HD-Charon", "ja-JP", "ja-JP"},
		{"empty voice uses the fallback", "", "ja-JP", "ja-JP"},
		{"nothing to go on uses the default", "", "", "en-US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.extractLangCode(tt.voiceID, tt.fallback); got != tt.expect {
				t.Errorf("Wrong language code. expected: %q, got: %q", tt.expect, got)
			}
		})
	}
}
