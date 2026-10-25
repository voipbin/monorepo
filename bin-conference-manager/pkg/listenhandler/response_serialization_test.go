package listenhandler

import (
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

// Test_processRequest_responseSerializationFailure verifies the external response
// code of every conference route whose handler result cannot be serialized.
// The expected code differs per route and is pinned as it is currently returned.
func Test_processRequest_responseSerializationFailure(t *testing.T) {
	const base = "/v1/conferences/" + testErrPathID

	unserializable := func() *conference.Conference {
		return &conference.Conference{Data: map[string]any{"bad": make(chan int)}}
	}

	tests := []struct {
		name    string
		request *sock.Request

		setup      func(cf *conferencehandler.MockConferenceHandler)
		expectCode int
	}{
		{
			name:    "conferences list",
			request: &sock.Request{URI: "/v1/conferences?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*conference.Conference{unserializable()}, nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferences create",
			request: &sock.Request{URI: "/v1/conferences", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conference get",
			request: &sock.Request{URI: base, Method: sock.RequestMethodGet},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().Get(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "conference update",
			request: &sock.Request{URI: base, Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conference delete",
			request: &sock.Request{URI: base, Method: sock.RequestMethodDelete},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording id update",
			request: &sock.Request{URI: base + "/recording_id", Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().UpdateRecordingID(gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording start",
			request: &sock.Request{URI: base + "/recording_start", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().RecordingStart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording stop",
			request: &sock.Request{URI: base + "/recording_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().RecordingStop(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "transcribe start",
			request: &sock.Request{URI: base + "/transcribe_start", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().TranscribeStart(gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "transcribe stop",
			request: &sock.Request{URI: base + "/transcribe_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().TranscribeStop(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "stop",
			request: &sock.Request{URI: base + "/stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().Terminating(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "direct hash regenerate",
			request: &sock.Request{URI: base + "/direct-hash-regenerate", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler) {
				cf.EXPECT().DirectHashRegenerate(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			h := &listenHandler{
				sockHandler:           sockhandler.NewMockSockHandler(mc),
				conferenceHandler:     mockConf,
				conferencecallHandler: conferencecallhandler.NewMockConferencecallHandler(mc),
			}
			tt.setup(mockConf)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectCode {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectCode, res)
			}
		})
	}
}
