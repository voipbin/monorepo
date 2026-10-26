package listenhandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// Test_processV1_shortURI checks every handler that reads an id from the path answers 400 for a
// path without it, and never calls the business handler (no expectation is set on the mocks).
func Test_processV1_shortURI(t *testing.T) {
	tests := []struct {
		name string
		call func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)
		uri  string
	}{
		{"post", (*listenHandler).processV1TranscribesPost, "/v1"},
		{"id get", (*listenHandler).processV1TranscribesIDGet, "/v1/transcribes"},
		{"id delete", (*listenHandler).processV1TranscribesIDDelete, "/v1/transcribes"},
		{"id stop", (*listenHandler).processV1TranscribesIDStopPost, "/v1/transcribes"},
		{"id health check", (*listenHandler).processV1TranscribesIDHealthCheckPost, "/v1/transcribes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc),
				transcriptHandler: transcripthandler.NewMockTranscriptHandler(mc),
			}

			res, err := tt.call(h, context.Background(), &sock.Request{URI: tt.uri, Method: sock.RequestMethodPost, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("expected a response, got an error. err: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("status mismatch. expect: 400, got: %+v", res)
			}
		})
	}
}

// Test_processV1_listBadInput checks the list endpoints reject a malformed filter body with 400
// and report an unparsable uri as an error, without calling the business handler.
func Test_processV1_listBadInput(t *testing.T) {
	lists := []struct {
		name string
		call func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)
		uri  string
	}{
		{"transcribes", (*listenHandler).processV1TranscribesGet, "/v1/transcribes?page_size=10"},
		{"transcripts", (*listenHandler).processV1TranscriptsGet, "/v1/transcripts?page_size=10"},
	}

	for _, tt := range lists {
		t.Run(tt.name+" malformed filter body", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc),
				transcriptHandler: transcripthandler.NewMockTranscriptHandler(mc),
			}

			res, err := tt.call(h, context.Background(), &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":`)})
			if err != nil {
				t.Fatalf("expected a response, got an error. err: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("status mismatch. expect: 400, got: %+v", res)
			}
		})

		t.Run(tt.name+" unparsable uri", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc),
				transcriptHandler: transcripthandler.NewMockTranscriptHandler(mc),
			}

			res, err := tt.call(h, context.Background(), &sock.Request{URI: "://bad", Method: sock.RequestMethodGet})
			if err == nil {
				t.Errorf("expected an error for an unparsable uri. got: %+v", res)
			}
		})
	}
}

// Test_processV1TranscribesPost_malformedBody checks a body that is not json answers 400.
func Test_processV1TranscribesPost_malformedBody(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc),
	}

	res, err := h.processV1TranscribesPost(context.Background(), &sock.Request{URI: "/v1/transcribes", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":`)})
	if err != nil {
		t.Fatalf("expected a response, got an error. err: %v", err)
	}
	if res == nil || res.StatusCode != 400 {
		t.Errorf("status mismatch. expect: 400, got: %+v", res)
	}
}
