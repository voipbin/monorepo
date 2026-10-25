package listenhandler

import (
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/speakinghandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	testIDString = "b6dcbc72-5af5-11f0-b0a7-fbe8aae41529"
	// brokenPrefix makes url.Parse fail while the routing regexes (not anchored at the start) still match.
	brokenPrefix = "/%zz"
)

var errTestDownstream = errors.New("downstream failure")

type errorPathMocks struct {
	speaking  *speakinghandler.MockSpeakingHandler
	streaming *streaminghandler.MockStreamingHandler
	tts       *ttshandler.MockTTSHandler
}

func Test_processRequest_speakingsErrorPaths(t *testing.T) {
	testID := uuid.FromStringOrNil(testIDString)

	tests := []struct {
		name string

		request *sock.Request
		setup   func(m *errorPathMocks)

		expectStatus int
	}{
		{
			name:         "post with invalid json",
			request:      &sock.Request{URI: "/v1/speakings", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "post with create failure",
			request: &sock.Request{URI: "/v1/speakings", Method: sock.RequestMethodPost, Data: []byte(`{"language":"en-US"}`)},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Create(gomock.Any(), uuid.Nil, streaming.ReferenceType(""), uuid.Nil, "en-US", "", "", streaming.Direction("")).Return(nil, errTestDownstream)
			},
			expectStatus: 500,
		},
		{
			name:         "get list with malformed uri",
			request:      &sock.Request{URI: brokenPrefix + "/v1/speakings", Method: sock.RequestMethodGet},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "get list with lookup failure",
			request: &sock.Request{URI: "/v1/speakings?page_size=5", Method: sock.RequestMethodGet},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Gets(gomock.Any(), "", uint64(5), gomock.Any()).Return(nil, errTestDownstream)
			},
			expectStatus: 500,
		},
		{
			name:         "get by id with malformed uri",
			request:      &sock.Request{URI: brokenPrefix + "/v1/speakings/" + testIDString, Method: sock.RequestMethodGet},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "get by id with lookup failure",
			request: &sock.Request{URI: "/v1/speakings/" + testIDString, Method: sock.RequestMethodGet},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Get(gomock.Any(), testID).Return(nil, errTestDownstream)
			},
			expectStatus: 404,
		},
		{
			name:         "delete with malformed uri",
			request:      &sock.Request{URI: brokenPrefix + "/v1/speakings/" + testIDString, Method: sock.RequestMethodDelete},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "delete with failure",
			request: &sock.Request{URI: "/v1/speakings/" + testIDString, Method: sock.RequestMethodDelete},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Delete(gomock.Any(), testID).Return(nil, errTestDownstream)
			},
			expectStatus: 404,
		},
		{
			name:         "say with malformed uri",
			request:      &sock.Request{URI: brokenPrefix + "/v1/speakings/" + testIDString + "/say", Method: sock.RequestMethodPost, Data: []byte(`{"text":"hi"}`)},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:         "say with invalid json",
			request:      &sock.Request{URI: "/v1/speakings/" + testIDString + "/say", Method: sock.RequestMethodPost, Data: []byte(`not json`)},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "say with failure",
			request: &sock.Request{URI: "/v1/speakings/" + testIDString + "/say", Method: sock.RequestMethodPost, Data: []byte(`{"text":"hi"}`)},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Say(gomock.Any(), testID, "hi").Return(nil, errTestDownstream)
			},
			expectStatus: 404,
		},
		{
			name:         "flush with malformed uri",
			request:      &sock.Request{URI: brokenPrefix + "/v1/speakings/" + testIDString + "/flush", Method: sock.RequestMethodPost},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "flush with failure",
			request: &sock.Request{URI: "/v1/speakings/" + testIDString + "/flush", Method: sock.RequestMethodPost},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Flush(gomock.Any(), testID).Return(nil, errTestDownstream)
			},
			expectStatus: 404,
		},
		{
			name:         "stop with malformed uri",
			request:      &sock.Request{URI: brokenPrefix + "/v1/speakings/" + testIDString + "/stop", Method: sock.RequestMethodPost},
			setup:        func(m *errorPathMocks) {},
			expectStatus: 400,
		},
		{
			name:    "stop with failure",
			request: &sock.Request{URI: "/v1/speakings/" + testIDString + "/stop", Method: sock.RequestMethodPost},
			setup: func(m *errorPathMocks) {
				m.speaking.EXPECT().Stop(gomock.Any(), testID).Return(nil, errTestDownstream)
			},
			expectStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := &errorPathMocks{
				speaking:  speakinghandler.NewMockSpeakingHandler(mc),
				streaming: streaminghandler.NewMockStreamingHandler(mc),
				tts:       ttshandler.NewMockTTSHandler(mc),
			}
			h := &listenHandler{
				sockHandler:      sockhandler.NewMockSockHandler(mc),
				ttsHandler:       m.tts,
				streamingHandler: m.streaming,
				speakingHandler:  m.speaking,
			}
			tt.setup(m)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("expected a response without a transport error, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %v", tt.expectStatus, res)
			}
			if len(res.Data) != 0 {
				t.Errorf("Expected an empty body for an error response, got: %s", res.Data)
			}
		})
	}
}

func Test_v1SpeakingsGet_pageSize(t *testing.T) {
	tests := []struct {
		name string

		uri string

		expectSize uint64
	}{
		{name: "unset uses the default", uri: "/v1/speakings", expectSize: 100},
		{name: "zero uses the default", uri: "/v1/speakings?page_size=0", expectSize: 100},
		{name: "above the limit is clamped", uri: "/v1/speakings?page_size=1000", expectSize: 100},
		{name: "not a number uses the default", uri: "/v1/speakings?page_size=abc", expectSize: 100},
		{name: "within the limit is kept", uri: "/v1/speakings?page_size=5", expectSize: 5},
		{name: "exact limit is kept", uri: "/v1/speakings?page_size=100", expectSize: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)
			h := &listenHandler{speakingHandler: mockSpeaking}

			mockSpeaking.EXPECT().Gets(gomock.Any(), "", tt.expectSize, gomock.Any()).Return([]*speaking.Speaking{}, nil)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodGet})
			if err != nil || res == nil || res.StatusCode != 200 {
				t.Errorf("Wrong result. res: %v, err: %v", res, err)
			}
		})
	}
}

func Test_v1SpeakingsGet_filters(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)
	h := &listenHandler{speakingHandler: mockSpeaking}

	referenceID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	expectFilters := map[speaking.Field]any{
		speaking.FieldDeleted:       false,
		speaking.FieldReferenceType: "call",
		speaking.FieldReferenceID:   referenceID,
	}

	mockSpeaking.EXPECT().Gets(gomock.Any(), "token1", uint64(10), expectFilters).Return([]*speaking.Speaking{}, nil)

	uri := "/v1/speakings?page_token=token1&page_size=10&reference_type=call&reference_id=" + referenceID.String()
	res, err := h.processRequest(&sock.Request{URI: uri, Method: sock.RequestMethodGet})
	if err != nil || res == nil || res.StatusCode != 200 {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
}

func Test_processRequest_streamingsErrorPaths(t *testing.T) {
	testID := uuid.FromStringOrNil(testIDString)
	messageID := uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	tests := []struct {
		name string

		request *sock.Request
		setup   func(m *errorPathMocks)

		expectErr error
	}{
		{
			name:    "start with invalid json",
			request: &sock.Request{URI: "/v1/streamings", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "start with failure",
			request: &sock.Request{URI: "/v1/streamings", Method: sock.RequestMethodPost, Data: []byte(`{"language":"en-US"}`)},
			setup: func(m *errorPathMocks) {
				m.streaming.EXPECT().Start(gomock.Any(), uuid.Nil, uuid.Nil, streaming.ReferenceType(""), uuid.Nil, "en-US", streaming.Gender(""), streaming.Direction("")).Return(nil, errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
		{
			name:    "delete with malformed uri",
			request: &sock.Request{URI: brokenPrefix + "/v1/streamings/" + testIDString, Method: sock.RequestMethodDelete},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "delete with failure",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString, Method: sock.RequestMethodDelete},
			setup: func(m *errorPathMocks) {
				m.streaming.EXPECT().Stop(gomock.Any(), testID).Return(nil, errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
		{
			name:    "say init with malformed uri",
			request: &sock.Request{URI: brokenPrefix + "/v1/streamings/" + testIDString + "/say_init", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say init with invalid json",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_init", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say init with failure",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_init", Method: sock.RequestMethodPost, Data: []byte(`{"message_id":"` + messageID.String() + `"}`)},
			setup: func(m *errorPathMocks) {
				m.streaming.EXPECT().SayInit(gomock.Any(), testID, messageID).Return(nil, errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
		{
			name:    "say add with malformed uri",
			request: &sock.Request{URI: brokenPrefix + "/v1/streamings/" + testIDString + "/say_add", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say add with invalid json",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_add", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say add with failure",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_add", Method: sock.RequestMethodPost, Data: []byte(`{"message_id":"` + messageID.String() + `","text":"hello"}`)},
			setup: func(m *errorPathMocks) {
				m.streaming.EXPECT().SayAdd(gomock.Any(), testID, messageID, "hello").Return(errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
		{
			name:    "say stop with malformed uri",
			request: &sock.Request{URI: brokenPrefix + "/v1/streamings/" + testIDString + "/say_stop", Method: sock.RequestMethodPost},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say stop with failure",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_stop", Method: sock.RequestMethodPost},
			setup: func(m *errorPathMocks) {
				m.streaming.EXPECT().SayStop(gomock.Any(), testID).Return(errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
		{
			name:    "say finish with malformed uri",
			request: &sock.Request{URI: brokenPrefix + "/v1/streamings/" + testIDString + "/say_finish", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say finish with invalid json",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_finish", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "say finish with failure",
			request: &sock.Request{URI: "/v1/streamings/" + testIDString + "/say_finish", Method: sock.RequestMethodPost, Data: []byte(`{"message_id":"` + messageID.String() + `"}`)},
			setup: func(m *errorPathMocks) {
				m.streaming.EXPECT().SayFinish(gomock.Any(), testID, messageID).Return(nil, errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
		{
			name:    "speeches with invalid json",
			request: &sock.Request{URI: "/v1/speeches", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
			setup:   func(m *errorPathMocks) {},
		},
		{
			name:    "speeches with create failure",
			request: &sock.Request{URI: "/v1/speeches", Method: sock.RequestMethodPost, Data: []byte(`{"text":"hello","language":"en-US"}`)},
			setup: func(m *errorPathMocks) {
				m.tts.EXPECT().Create(gomock.Any(), uuid.Nil, "hello", "en-US", gomock.Any(), "").Return(nil, errTestDownstream)
			},
			expectErr: errTestDownstream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := &errorPathMocks{
				speaking:  speakinghandler.NewMockSpeakingHandler(mc),
				streaming: streaminghandler.NewMockStreamingHandler(mc),
				tts:       ttshandler.NewMockTTSHandler(mc),
			}
			h := &listenHandler{
				sockHandler:      sockhandler.NewMockSockHandler(mc),
				ttsHandler:       m.tts,
				streamingHandler: m.streaming,
				speakingHandler:  m.speaking,
			}
			tt.setup(m)

			res, err := h.processRequest(tt.request)
			if err == nil {
				t.Fatalf("expected an error to be propagated, got response: %v", res)
			}
			if res != nil {
				t.Errorf("expected no response with an error, got: %v", res)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processRequest_errorConversion(t *testing.T) {
	typedErr := cerrors.NotFound("tts-manager", "TEST_NOT_FOUND", "not found for test")
	expectTyped, errConvert := cerrors.ToResponse(typedErr)
	if errConvert != nil {
		t.Fatalf("could not build the expected typed response: %v", errConvert)
	}

	tests := []struct {
		name string

		returnErr error

		expectRes *sock.Response
		expectErr error
	}{
		{
			name:      "typed error becomes a response",
			returnErr: typedErr,
			expectRes: expectTyped,
		},
		{
			name:      "db not found becomes 404",
			returnErr: dbhandler.ErrNotFound,
			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name:      "other error is propagated",
			returnErr: errTestDownstream,
			expectErr: errTestDownstream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
			h := &listenHandler{streamingHandler: mockStreaming}

			mockStreaming.EXPECT().SayStop(gomock.Any(), gomock.Any()).Return(tt.returnErr)

			res, err := h.processRequest(&sock.Request{URI: "/v1/streamings/" + testIDString + "/say_stop", Method: sock.RequestMethodPost})

			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Errorf("Wrong result. expect error: %v, got res: %v, err: %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected the error to be converted into a response, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectRes.StatusCode || string(res.Data) != string(tt.expectRes.Data) {
				t.Errorf("Wrong response.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue creation failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errTestDownstream)

		err := h.Run("test-queue", "")
		if !errors.Is(err, errTestDownstream) {
			t.Errorf("Wrong error. expect: %v, got: %v", errTestDownstream, err)
		}
	})

	t.Run("consumer is started on success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		consumed := make(chan string, 1)
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "tts-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
			func(_ any, queue string, _ string, _, _, _ bool, _ int, _ any) error {
				consumed <- queue
				return nil
			})

		if err := h.Run("test-queue", ""); err != nil {
			t.Fatalf("expected nil, got: %v", err)
		}

		select {
		case got := <-consumed:
			if got != "test-queue" {
				t.Errorf("Wrong queue. expect: test-queue, got: %s", got)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the consumer was not started")
		}
	})
}
