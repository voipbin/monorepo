package listenhandler

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// Test_errorResponse_unmarshalableDetails checks that a typed error whose detail cannot be
// encoded is answered with a plain 500 and without a body, not with a partial typed error.
func Test_errorResponse_unmarshalableDetails(t *testing.T) {
	ve := cerrors.NotFound(commonoutline.ServiceNameTranscribeManager, "TRANSCRIBE_NOT_FOUND", "x")
	ve.Details = []map[string]any{{"unsupported": make(chan int)}}

	res := errorResponse(ve)
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
	if res.DataType != "" || len(res.Data) != 0 {
		t.Errorf("Wrong match. expect: an empty body, got: %q %q", res.DataType, res.Data)
	}
}

// Test_process_responseMarshalFailure checks that every handler that answers with the encoded
// result reports a 500 response, not a transport error, when the result cannot be encoded. The
// time stamp of the mocked result is outside of the range that JSON can represent.
func Test_process_responseMarshalFailure(t *testing.T) {
	id := uuid.FromStringOrNil("b25a1e10-0000-11ee-9c11-000000000011")
	customerID := uuid.FromStringOrNil("b25a1e10-0000-11ee-9c11-000000000012")
	referenceID := uuid.FromStringOrNil("b25a1e10-0000-11ee-9c11-000000000013")
	badTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	newTranscribe := func() *transcribe.Transcribe {
		return &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}, TMCreate: &badTime}
	}

	tests := []struct {
		name string

		request *sock.Request
		setup   func(mockTranscribe *transcribehandler.MockTranscribeHandler, mockTranscript *transcripthandler.MockTranscriptHandler)
	}{
		{
			name: "start",
			request: &sock.Request{
				URI:    "/v1/transcribes",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id":"` + customerID.String() + `","reference_type":"call","reference_id":"` + referenceID.String() + `","language":"en-US","direction":"both"}`),
			},
			setup: func(mockTranscribe *transcribehandler.MockTranscribeHandler, _ *transcripthandler.MockTranscriptHandler) {
				mockTranscribe.EXPECT().Start(gomock.Any(), uuid.Nil, customerID, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, referenceID, "en-US", transcribe.DirectionBoth, transcribe.ProviderEmpty).Return(newTranscribe(), nil)
			},
		},
		{
			name: "list",
			request: &sock.Request{
				URI:    "/v1/transcribes?page_size=10&page_token=2020-05-03T21:35:02.809Z",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			setup: func(mockTranscribe *transcribehandler.MockTranscribeHandler, _ *transcripthandler.MockTranscriptHandler) {
				mockTranscribe.EXPECT().List(gomock.Any(), uint64(10), "2020-05-03T21:35:02.809Z", gomock.Any()).Return([]*transcribe.Transcribe{newTranscribe()}, nil)
			},
		},
		{
			name:    "get",
			request: &sock.Request{URI: "/v1/transcribes/" + id.String(), Method: sock.RequestMethodGet},
			setup: func(mockTranscribe *transcribehandler.MockTranscribeHandler, _ *transcripthandler.MockTranscriptHandler) {
				mockTranscribe.EXPECT().Get(gomock.Any(), id).Return(newTranscribe(), nil)
			},
		},
		{
			name:    "delete",
			request: &sock.Request{URI: "/v1/transcribes/" + id.String(), Method: sock.RequestMethodDelete},
			setup: func(mockTranscribe *transcribehandler.MockTranscribeHandler, _ *transcripthandler.MockTranscriptHandler) {
				mockTranscribe.EXPECT().Delete(gomock.Any(), id).Return(newTranscribe(), nil)
			},
		},
		{
			name:    "stop",
			request: &sock.Request{URI: "/v1/transcribes/" + id.String() + "/stop", Method: sock.RequestMethodPost},
			setup: func(mockTranscribe *transcribehandler.MockTranscribeHandler, _ *transcripthandler.MockTranscriptHandler) {
				mockTranscribe.EXPECT().Stop(gomock.Any(), id).Return(newTranscribe(), nil)
			},
		},
		{
			name: "transcripts list",
			request: &sock.Request{
				URI:    "/v1/transcripts?page_size=10&page_token=2020-05-03T21:35:02.809Z",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{}`),
			},
			setup: func(_ *transcribehandler.MockTranscribeHandler, mockTranscript *transcripthandler.MockTranscriptHandler) {
				mockTranscript.EXPECT().List(gomock.Any(), uint64(10), "2020-05-03T21:35:02.809Z", gomock.Any()).Return([]*transcript.Transcript{{Identity: commonidentity.Identity{ID: id}, TMCreate: &badTime}}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			h := &listenHandler{transcribeHandler: mockTranscribe, transcriptHandler: mockTranscript}

			tt.setup(mockTranscribe, mockTranscript)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusInternalServerError {
				t.Errorf("Wrong match. expect: %d, got: %v", http.StatusInternalServerError, res)
			}
			if res != nil && len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: an empty body, got: %q", res.Data)
			}
		})
	}
}
