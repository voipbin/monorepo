package listenhandler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	pkgerrors "github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

const testErrPathID = "8d920096-3bf2-11ec-9ff1-87ad93d2f885"

// Test_processRequest_errorPaths verifies the external response code of every
// failure branch of the v1 handlers when they are reached through processRequest.
// The mock controllers have no expectation for the calls that must not happen,
// so any call after an input validation failure fails the test.
func Test_processRequest_errorPaths(t *testing.T) {
	errGeneric := errors.New("simulated failure")
	errTyped := cerrors.NotFound(commonoutline.ServiceNameConferenceManager, "CONFERENCE_NOT_FOUND", "The conference was not found.")
	errWrappedNotFound := pkgerrors.Wrap(dbhandler.ErrNotFound, "could not get the conference")
	id := uuid.FromStringOrNil(testErrPathID)

	tests := []struct {
		name    string
		request *sock.Request

		setup      func(cf *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler)
		expectCode int
	}{
		// unknown route
		{
			name:       "unknown uri",
			request:    &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			expectCode: http.StatusNotFound,
		},
		{
			name:       "known uri with unsupported method",
			request:    &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/stop", Method: sock.RequestMethodGet},
			expectCode: http.StatusNotFound,
		},

		// conferences list
		{
			name:       "conferences get: url.Parse failure",
			request:    &sock.Request{URI: "/x%zz/v1/conferences?page_size=1", Method: sock.RequestMethodGet},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "conferences get: filter body is not an object",
			request:    &sock.Request{URI: "/v1/conferences?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`[1]`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "conferences get: filter value cannot be converted",
			request:    &sock.Request{URI: "/v1/conferences?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "conferences get: list returns a generic error",
			request: &sock.Request{URI: "/v1/conferences?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().List(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferences get: list returns a typed error",
			request: &sock.Request{URI: "/v1/conferences?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().List(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, errTyped)
			},
			expectCode: http.StatusNotFound,
		},

		// conferences create
		{
			name:       "conferences post: broken json",
			request:    &sock.Request{URI: "/v1/conferences", Method: sock.RequestMethodPost, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "conferences post: create returns a generic error",
			request: &sock.Request{URI: "/v1/conferences", Method: sock.RequestMethodPost, Data: []byte(`{"type":"conference"}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.Nil, conference.TypeConference, "", "", nil, 0, uuid.Nil, uuid.Nil).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferences post: create returns not found",
			request: &sock.Request{URI: "/v1/conferences", Method: sock.RequestMethodPost, Data: []byte(`{"type":"conference"}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.Nil, conference.TypeConference, "", "", nil, 0, uuid.Nil, uuid.Nil).Return(nil, errWrappedNotFound)
			},
			expectCode: http.StatusNotFound,
		},

		// conferences id
		{
			name:    "conferences delete: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodDelete},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Delete(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferences get id: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodGet},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Get(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferences delete: typed not found",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodDelete},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Delete(gomock.Any(), id).Return(nil, errTyped)
			},
			expectCode: http.StatusNotFound,
		},
		{
			name:    "conferences get id: typed not found",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodGet},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Get(gomock.Any(), id).Return(nil, errTyped)
			},
			expectCode: http.StatusNotFound,
		},
		{
			name:    "conferences put: wrapped not found",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"n"}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Update(gomock.Any(), id, gomock.Any(), nil, nil, nil, nil, nil).Return(nil, errWrappedNotFound)
			},
			expectCode: http.StatusNotFound,
		},
		{
			name:       "conferences put: broken json",
			request:    &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodPut, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "conferences put: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"n"}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Update(gomock.Any(), id, gomock.Any(), nil, nil, nil, nil, nil).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},

		// recording
		{
			name:       "recording_id put: broken json",
			request:    &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_id", Method: sock.RequestMethodPut, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording_id put: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_id", Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().UpdateRecordingID(gomock.Any(), id, uuid.Nil).Return(nil, errGeneric)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording_id put: handler error wrapping not found",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_id", Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().UpdateRecordingID(gomock.Any(), id, uuid.Nil).Return(nil, errWrappedNotFound)
			},
			expectCode: http.StatusNotFound,
		},
		{
			name:       "recording_start post: broken json",
			request:    &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_start", Method: sock.RequestMethodPost, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording_start post: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_start", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().RecordingStart(gomock.Any(), id, uuid.Nil, gomock.Any(), 0, uuid.Nil).Return(nil, errGeneric)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording_stop post: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().RecordingStop(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "recording_stop post: handler error with typed error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/recording_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().RecordingStop(gomock.Any(), id).Return(nil, errTyped)
			},
			expectCode: http.StatusNotFound,
		},

		// transcribe
		{
			name:       "transcribe_start post: broken json",
			request:    &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/transcribe_start", Method: sock.RequestMethodPost, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "transcribe_start post: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/transcribe_start", Method: sock.RequestMethodPost, Data: []byte(`{"language":"en-US"}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().TranscribeStart(gomock.Any(), id, "en-US").Return(nil, errGeneric)
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "transcribe_stop post: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/transcribe_stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().TranscribeStop(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusBadRequest,
		},

		// stop
		{
			name:    "stop post: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/stop", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().Terminating(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusBadRequest,
		},

		// count_by_customer
		{
			name:       "count_by_customer: broken json",
			request:    &sock.Request{URI: "/v1/conferences/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "count_by_customer: handler error",
			request: &sock.Request{URI: "/v1/conferences/count_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().CountByCustomerID(gomock.Any(), uuid.Nil).Return(0, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},

		// direct hash
		{
			name:    "direct-hash-regenerate: handler error",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/direct-hash-regenerate", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().DirectHashRegenerate(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "direct-hash-regenerate: not found",
			request: &sock.Request{URI: "/v1/conferences/" + testErrPathID + "/direct-hash-regenerate", Method: sock.RequestMethodPost},
			setup: func(cf *conferencehandler.MockConferenceHandler, _ *conferencecallhandler.MockConferencecallHandler) {
				cf.EXPECT().DirectHashRegenerate(gomock.Any(), id).Return(nil, errWrappedNotFound)
			},
			expectCode: http.StatusNotFound,
		},

		// conferencecalls
		{
			name:       "conferencecalls get: url.Parse failure",
			request:    &sock.Request{URI: "/x%zz/v1/conferencecalls?page_size=1", Method: sock.RequestMethodGet},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "conferencecalls get: filter body is not an object",
			request:    &sock.Request{URI: "/v1/conferencecalls?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`[1]`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "conferencecalls get: filter value cannot be converted",
			request:    &sock.Request{URI: "/v1/conferencecalls?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "conferencecalls get: list returns a generic error",
			request: &sock.Request{URI: "/v1/conferencecalls?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().List(gomock.Any(), uint64(1), "", gomock.Any()).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferencecalls get id: handler error",
			request: &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID, Method: sock.RequestMethodGet},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().Get(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferencecalls delete: handler error",
			request: &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID, Method: sock.RequestMethodDelete},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().Terminate(gomock.Any(), id).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "conferencecalls get id: typed not found",
			request: &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID, Method: sock.RequestMethodGet},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().Get(gomock.Any(), id).Return(nil, errTyped)
			},
			expectCode: http.StatusNotFound,
		},
		{
			name:    "conferencecalls delete: wrapped not found",
			request: &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID, Method: sock.RequestMethodDelete},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().Terminate(gomock.Any(), id).Return(nil, errWrappedNotFound)
			},
			expectCode: http.StatusNotFound,
		},
		{
			name:       "conferencecalls health-check: broken json",
			request:    &sock.Request{URI: "/v1/conferencecalls/" + testErrPathID + "/health-check", Method: sock.RequestMethodPost, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},

		// services
		{
			name:       "services conferencecall post: broken json",
			request:    &sock.Request{URI: "/v1/services/type/conferencecall", Method: sock.RequestMethodPost, Data: []byte(`{`)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:    "services conferencecall post: service start error",
			request: &sock.Request{URI: "/v1/services/type/conferencecall", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().ServiceStart(gomock.Any(), uuid.Nil, uuid.Nil, gomock.Any(), uuid.Nil).Return(nil, errGeneric)
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name:    "services conferencecall post: service start typed error",
			request: &sock.Request{URI: "/v1/services/type/conferencecall", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup: func(_ *conferencehandler.MockConferenceHandler, cc *conferencecallhandler.MockConferencecallHandler) {
				cc.EXPECT().ServiceStart(gomock.Any(), uuid.Nil, uuid.Nil, gomock.Any(), uuid.Nil).Return(nil, errTyped)
			},
			expectCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockCC := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				conferenceHandler:     mockConf,
				conferencecallHandler: mockCC,
			}

			if tt.setup != nil {
				tt.setup(mockConf, mockCC)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Fatalf("Wrong match. expect: response, got: nil")
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("Wrong status code.\nexpect: %d\ngot: %d", tt.expectCode, res.StatusCode)
			}
		})
	}
}

// Test_processV1_shortURI verifies that the id based handlers reject a uri with
// too few path items before calling any business handler.
func Test_processV1_shortURI(t *testing.T) {
	// uri items are counted after splitting by "/". A uri without the leading
	// slash has one item less than a routed one.
	id := testErrPathID

	tests := []struct {
		name string
		uri  string
		call func(h *listenHandler, m *sock.Request) (*sock.Response, error)
	}{
		{"conferences delete", "v1/conferences/" + id, func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDDelete(context.Background(), m)
		}},
		{"conferences put", "v1/conferences/" + id, func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDPut(context.Background(), m)
		}},
		{"conferences get id", "v1/conferences/" + id, func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDGet(context.Background(), m)
		}},
		{"recording_id put", "v1/conferences/" + id + "/recording_id", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDRecordingIDPut(context.Background(), m)
		}},
		{"recording_start post", "v1/conferences/" + id + "/recording_start", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDRecordingStartPost(context.Background(), m)
		}},
		{"recording_stop post", "v1/conferences/" + id + "/recording_stop", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDRecordingStopPost(context.Background(), m)
		}},
		{"transcribe_start post", "v1/conferences/" + id + "/transcribe_start", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDTranscribeStartPost(context.Background(), m)
		}},
		{"transcribe_stop post", "v1/conferences/" + id + "/transcribe_stop", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDTranscribeStopPost(context.Background(), m)
		}},
		{"stop post", "v1/conferences/" + id + "/stop", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDStopPost(context.Background(), m)
		}},
		{"direct-hash-regenerate", "v1/conferences", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencesIDDirectHashRegenerate(context.Background(), m)
		}},
		{"conferencecalls get id", "v1/conferencecalls/" + id, func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencecallsIDGet(context.Background(), m)
		}},
		{"conferencecalls delete", "v1/conferencecalls/" + id, func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencecallsIDDelete(context.Background(), m)
		}},
		{"conferencecalls health-check", "v1/conferencecalls/" + id, func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1ConferencecallsIDHealthCheckPost(context.Background(), m)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				sockHandler:           sockhandler.NewMockSockHandler(mc),
				conferenceHandler:     conferencehandler.NewMockConferenceHandler(mc),
				conferencecallHandler: conferencecallhandler.NewMockConferencecallHandler(mc),
			}

			res, err := tt.call(h, &sock.Request{URI: tt.uri, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

func Test_processRequest_directHashRegenerate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockConf := conferencehandler.NewMockConferenceHandler(mc)
	h := &listenHandler{
		sockHandler:       sockhandler.NewMockSockHandler(mc),
		conferenceHandler: mockConf,
	}

	id := uuid.FromStringOrNil(testErrPathID)
	mockConf.EXPECT().DirectHashRegenerate(gomock.Any(), id).Return(&conference.Conference{DirectHash: "abc123"}, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/conferences/" + testErrPathID + "/direct-hash-regenerate",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
	}
	if !strings.Contains(string(res.Data), `"direct_hash":"abc123"`) {
		t.Errorf("Wrong body. got: %s", res.Data)
	}
}
