package listenhandler

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_processRequest_errorResponses(t *testing.T) {
	id := "a14f0d10-0000-11ee-8c11-000000000001"
	idUUID := uuid.FromStringOrNil(id)
	notFound := cerrors.NotFound(commonoutline.ServiceNameTranscribeManager, "TRANSCRIBE_NOT_FOUND", "not found")

	tests := []struct {
		name string

		request *sock.Request

		// expectations on the business handlers, applied only when set
		expectGet    error
		expectDelete error
		expectStop   error
		expectList   error
		expectStart  error
		expectTscrpt error

		expectStatus int
	}{
		{
			name:         "get transcribe: typed not found maps to 404",
			request:      &sock.Request{URI: "/v1/transcribes/" + id, Method: sock.RequestMethodGet},
			expectGet:    notFound,
			expectStatus: 404,
		},
		{
			name:         "get transcribe: legacy not found maps to 404",
			request:      &sock.Request{URI: "/v1/transcribes/" + id, Method: sock.RequestMethodGet},
			expectGet:    dbhandler.ErrNotFound,
			expectStatus: 404,
		},
		{
			name:         "get transcribe: unclassified error maps to 500",
			request:      &sock.Request{URI: "/v1/transcribes/" + id, Method: sock.RequestMethodGet},
			expectGet:    fmt.Errorf("db down"),
			expectStatus: 500,
		},
		{
			name:         "delete transcribe: typed not found maps to 404",
			request:      &sock.Request{URI: "/v1/transcribes/" + id, Method: sock.RequestMethodDelete},
			expectDelete: notFound,
			expectStatus: 404,
		},
		{
			name:         "stop transcribe: unclassified error maps to 500",
			request:      &sock.Request{URI: "/v1/transcribes/" + id + "/stop", Method: sock.RequestMethodPost},
			expectStop:   fmt.Errorf("stop failed"),
			expectStatus: 500,
		},
		{
			name:         "create transcribe: business error maps through the error response",
			request:      &sock.Request{URI: "/v1/transcribes", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"a14f0d10-0000-11ee-8c11-000000000002","reference_type":"call","reference_id":"a14f0d10-0000-11ee-8c11-000000000003","language":"en-US"}`)},
			expectStart:  notFound,
			expectStatus: 404,
		},
		{
			name:         "list transcribes: handler error maps to 500",
			request:      &sock.Request{URI: "/v1/transcribes?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expectList:   fmt.Errorf("list failed"),
			expectStatus: 500,
		},
		{
			name:         "list transcribes: malformed filter body is a 400",
			request:      &sock.Request{URI: "/v1/transcribes?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{not-json`)},
			expectStatus: 400,
		},
		{
			name:         "list transcribes: filter value that is not a valid uuid is a 400",
			request:      &sock.Request{URI: "/v1/transcribes?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectStatus: 400,
		},
		{
			name:         "list transcripts: handler error maps to 500",
			request:      &sock.Request{URI: "/v1/transcripts?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expectTscrpt: fmt.Errorf("list failed"),
			expectStatus: 500,
		},
		{
			name:         "list transcripts: malformed filter body is a 400",
			request:      &sock.Request{URI: "/v1/transcripts?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{not-json`)},
			expectStatus: 400,
		},
		{
			name:         "list transcripts: filter value that is not a valid uuid is a 400",
			request:      &sock.Request{URI: "/v1/transcripts?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectStatus: 400,
		},
		{
			name:         "health check with a malformed body is a 400",
			request:      &sock.Request{URI: "/v1/transcribes/" + id + "/health-check", Method: sock.RequestMethodPost, Data: []byte(`{not-json`)},
			expectStatus: 400,
		},
		{
			name:         "unknown path is a 404",
			request:      &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			expectStatus: 404,
		},
		{
			name:         "known path with an unsupported method is a 404",
			request:      &sock.Request{URI: "/v1/transcripts?page_size=10", Method: sock.RequestMethodDelete},
			expectStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			h := &listenHandler{transcribeHandler: mockTranscribe, transcriptHandler: mockTranscript}

			if tt.expectGet != nil {
				mockTranscribe.EXPECT().Get(gomock.Any(), idUUID).Return(nil, tt.expectGet)
			}
			if tt.expectDelete != nil {
				mockTranscribe.EXPECT().Delete(gomock.Any(), idUUID).Return(nil, tt.expectDelete)
			}
			if tt.expectStop != nil {
				mockTranscribe.EXPECT().Stop(gomock.Any(), idUUID).Return(nil, tt.expectStop)
			}
			if tt.expectStart != nil {
				mockTranscribe.EXPECT().Start(gomock.Any(), uuid.Nil, uuid.FromStringOrNil("a14f0d10-0000-11ee-8c11-000000000002"), uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.FromStringOrNil("a14f0d10-0000-11ee-8c11-000000000003"), "en-US", transcribe.Direction(""), transcribe.Provider("")).Return(nil, tt.expectStart)
			}
			if tt.expectList != nil {
				mockTranscribe.EXPECT().List(gomock.Any(), uint64(10), "", map[transcribe.Field]any{}).Return(nil, tt.expectList)
			}
			if tt.expectTscrpt != nil {
				mockTranscript.EXPECT().List(gomock.Any(), uint64(10), "", map[transcript.Field]any{}).Return(nil, tt.expectTscrpt)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect status: %d, got: %v", tt.expectStatus, res)
			}
		})
	}
}
