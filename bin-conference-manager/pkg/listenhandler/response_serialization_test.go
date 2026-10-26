package listenhandler

import (
	"net/http"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

// Test_processRequest_responseSerializationFailure verifies that every conference
// and conferencecall route answers with 500 when the handler result cannot be serialized.
func Test_processRequest_responseSerializationFailure(t *testing.T) {
	const base = "/v1/conferences/" + testErrPathID

	unserializable := func() *conference.Conference {
		return &conference.Conference{Data: map[string]any{"bad": make(chan int)}}
	}

	unserializableCall := func() *conferencecall.Conferencecall {
		tm := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		return &conferencecall.Conferencecall{TMCreate: &tm}
	}

	tests := []struct {
		name    string
		request *sock.Request

		setup      func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler)
		expectCode int
	}{
		{
			name:    "conferences list",
			request: &sock.Request{URI: "/v1/conferences?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*conference.Conference{unserializable()}, nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferences create",
			request: &sock.Request{URI: "/v1/conferences", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conference get",
			request: &sock.Request{URI: base, Method: sock.RequestMethodGet},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Get(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conference update",
			request: &sock.Request{URI: base, Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conference delete",
			request: &sock.Request{URI: base, Method: sock.RequestMethodDelete},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "recording id update",
			request: &sock.Request{URI: base + "/recording_id", Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().UpdateRecordingID(gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "recording start",
			request: &sock.Request{URI: base + "/recording_start", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().RecordingStart(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "recording stop",
			request: &sock.Request{URI: base + "/recording_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().RecordingStop(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "transcribe start",
			request: &sock.Request{URI: base + "/transcribe_start", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().TranscribeStart(gomock.Any(), gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "transcribe stop",
			request: &sock.Request{URI: base + "/transcribe_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().TranscribeStop(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "stop",
			request: &sock.Request{URI: base + "/stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Terminating(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "direct hash regenerate",
			request: &sock.Request{URI: base + "/direct-hash-regenerate", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().DirectHashRegenerate(gomock.Any(), gomock.Any()).Return(unserializable(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferencecall get",
			request: &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID, Method: sock.RequestMethodGet},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().Get(gomock.Any(), gomock.Any()).Return(unserializableCall(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferencecall delete",
			request: &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID, Method: sock.RequestMethodDelete},
			setup: func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().Terminate(gomock.Any(), gomock.Any()).Return(unserializableCall(), nil)
			},
			expectCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockCall := conferencecallhandler.NewMockConferencecallHandler(mc)
			h := &listenHandler{
				sockHandler:           sockhandler.NewMockSockHandler(mc),
				conferenceHandler:     mockConf,
				conferencecallHandler: mockCall,
			}
			tt.setup(mockConf, mockCall)

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
