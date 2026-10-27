package listenhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

const (
	testResID   = "7d1c9a3e-d13c-11ed-a1d2-0f3c5e6b7a88"
	testResID2  = "8e2dab4f-d13c-11ed-b2e3-1a4d6f7c8b99"
	testListURI = "?page_size=10"
)

type resourceMocks struct {
	confbridge     *confbridgehandler.MockConfbridgeHandler
	groupcall      *groupcallhandler.MockGroupcallHandler
	recording      *recordinghandler.MockRecordingHandler
	externalMedia  *externalmediahandler.MockExternalMediaHandler
	outboundConfig *outboundconfighandler.MockOutboundConfigHandler
}

func newResourceErrorHandler(mc *gomock.Controller) (*listenHandler, *resourceMocks) {
	m := &resourceMocks{
		confbridge:     confbridgehandler.NewMockConfbridgeHandler(mc),
		groupcall:      groupcallhandler.NewMockGroupcallHandler(mc),
		recording:      recordinghandler.NewMockRecordingHandler(mc),
		externalMedia:  externalmediahandler.NewMockExternalMediaHandler(mc),
		outboundConfig: outboundconfighandler.NewMockOutboundConfigHandler(mc),
	}
	return &listenHandler{
		sockHandler:           sockhandler.NewMockSockHandler(mc),
		confbridgeHandler:     m.confbridge,
		groupcallHandler:      m.groupcall,
		recordingHandler:      m.recording,
		externalMediaHandler:  m.externalMedia,
		outboundConfigHandler: m.outboundConfig,
	}, m
}

// Test_resources_shortURI verifies that a request URI that lacks the id segments is
// rejected with 400 before any business handler is called.
func Test_resources_shortURI(t *testing.T) {
	tests := []struct {
		name string
		fn   callsHandlerFunc
		uri  string
	}{
		{"confbridge get", (*listenHandler).processV1ConfbridgesIDGet, "/v1/confbridges"},
		{"confbridge delete", (*listenHandler).processV1ConfbridgesIDDelete, "/v1/confbridges"},
		{"confbridge terminate", (*listenHandler).processV1ConfbridgesIDTerminatePost, "/v1/confbridges"},
		{"confbridge kick", (*listenHandler).processV1ConfbridgesIDCallsIDDelete, "/v1/confbridges/" + testResID + "/calls"},
		{"confbridge join", (*listenHandler).processV1ConfbridgesIDCallsIDPost, "/v1/confbridges/" + testResID + "/calls"},
		{"confbridge external media start", (*listenHandler).processV1ConfbridgesIDExternalMediaPost, "/v1/confbridges"},
		{"confbridge external media stop", (*listenHandler).processV1ConfbridgesIDExternalMediaDelete, "/v1/confbridges"},
		{"confbridge recording start", (*listenHandler).processV1ConfbridgesIDRecordingStartPost, "/v1/confbridges"},
		{"confbridge recording stop", (*listenHandler).processV1ConfbridgesIDRecordingStopPost, "/v1/confbridges"},
		{"confbridge flag add", (*listenHandler).processV1ConfbridgesIDFlagsPost, "/v1/confbridges"},
		{"confbridge flag remove", (*listenHandler).processV1ConfbridgesIDFlagsDelete, "/v1/confbridges"},
		{"confbridge ring", (*listenHandler).processV1ConfbridgesIDRingPost, "/v1/confbridges"},
		{"confbridge answer", (*listenHandler).processV1ConfbridgesIDAnswerPost, "/v1/confbridges"},
		{"groupcall post", (*listenHandler).processV1GroupcallsPost, ""},
		{"groupcall get", (*listenHandler).processV1GroupcallsIDGet, "/v1/groupcalls"},
		{"groupcall delete", (*listenHandler).processV1GroupcallsIDDelete, "/v1/groupcalls"},
		{"groupcall hangup", (*listenHandler).processV1GroupcallsIDHangupPost, "/v1/groupcalls"},
		{"groupcall answer", (*listenHandler).processV1GroupcallsIDAnswerGroupcallIDPost, "/v1/groupcalls"},
		{"groupcall hangup groupcall", (*listenHandler).processV1GroupcallsIDHangupGroupcallPost, "/v1/groupcalls"},
		{"groupcall hangup call", (*listenHandler).processV1GroupcallsIDHangupCallPost, "/v1/groupcalls"},
		{"recording get", (*listenHandler).processV1RecordingsIDGet, "/v1/recordings"},
		{"recording delete", (*listenHandler).processV1RecordingsIDDelete, "/v1/recordings"},
		{"recording stop", (*listenHandler).processV1RecordingsIDStopPost, "/v1/recordings"},
		{"external media get", (*listenHandler).processV1ExternalMediasIDGet, "/v1/external-medias"},
		{"external media delete", (*listenHandler).processV1ExternalMediasIDDelete, "/v1/external-medias"},
		{"outbound config get", (*listenHandler).processV1OutboundConfigsIDGet, "/v1/outbound_configs"},
		{"outbound config delete", (*listenHandler).processV1OutboundConfigsIDDelete, "/v1/outbound_configs"},
		{"outbound config put", (*listenHandler).processV1OutboundConfigsIDPut, "/v1/outbound_configs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newResourceErrorHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{}`)})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

// Test_resources_invalidBody verifies the answer to a malformed JSON body.
func Test_resources_invalidBody(t *testing.T) {
	tests := []struct {
		name string
		fn   callsHandlerFunc
		uri  string

		expectBadRequest bool // 400 response, otherwise the decode error is returned
	}{
		{"confbridge create", (*listenHandler).processV1ConfbridgesPost, "/v1/confbridges", true},
		{"confbridge external media start", (*listenHandler).processV1ConfbridgesIDExternalMediaPost, "/v1/confbridges/" + testResID + "/external-media", false},
		{"confbridge external media stop", (*listenHandler).processV1ConfbridgesIDExternalMediaDelete, "/v1/confbridges/" + testResID + "/external-media", true},
		{"confbridge recording start", (*listenHandler).processV1ConfbridgesIDRecordingStartPost, "/v1/confbridges/" + testResID + "/recording_start", false},
		{"confbridge flag add", (*listenHandler).processV1ConfbridgesIDFlagsPost, "/v1/confbridges/" + testResID + "/flags", false},
		{"confbridge flag remove", (*listenHandler).processV1ConfbridgesIDFlagsDelete, "/v1/confbridges/" + testResID + "/flags", false},
		{"groupcall post", (*listenHandler).processV1GroupcallsPost, "/v1/groupcalls", true},
		{"groupcall answer", (*listenHandler).processV1GroupcallsIDAnswerGroupcallIDPost, "/v1/groupcalls/" + testResID + "/answer_groupcall_id", true},
		{"recording post", (*listenHandler).processV1RecordingsPost, "/v1/recordings", true},
		{"external media post", (*listenHandler).processV1ExternalMediasPost, "/v1/external-medias", true},
		{"outbound config post", (*listenHandler).processV1OutboundConfigsPost, "/v1/outbound_configs", true},
		{"outbound config put", (*listenHandler).processV1OutboundConfigsIDPut, "/v1/outbound_configs/" + testResID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newResourceErrorHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{not json`)})
			if tt.expectBadRequest {
				if err != nil || res == nil || res.StatusCode != 400 {
					t.Errorf("Wrong match. expect: 400, got: %v, %v", res, err)
				}
				return
			}
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: decode error, got: %v, %v", res, err)
			}
		})
	}
}

// Test_resources_listRequest verifies the list routes reject unusable queries and filters
// and report a failing business handler.
func Test_resources_listRequest(t *testing.T) {
	lists := []struct {
		name string
		fn   callsHandlerFunc
		base string
		bad  string // filter body whose value has a wrong type
		list func(m *resourceMocks)
	}{
		{"groupcalls", (*listenHandler).processV1GroupcallsGet, "/v1/groupcalls", `{"customer_id":"not-a-uuid"}`, func(m *resourceMocks) {
			m.groupcall.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return(nil, errTestBackend)
		}},
		{"recordings", (*listenHandler).processV1RecordingsGet, "/v1/recordings", `{"customer_id":"not-a-uuid"}`, func(m *resourceMocks) {
			m.recording.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return(nil, errTestBackend)
		}},
		{"external medias", (*listenHandler).processV1ExternalMediasGet, "/v1/external-medias", `{"id":"not-a-uuid"}`, func(m *resourceMocks) {
			m.externalMedia.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return(nil, errTestBackend)
		}},
	}

	for _, l := range lists {
		t.Run(l.name+" unparseable uri", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _ := newResourceErrorHandler(mc)

			res, err := l.fn(h, context.Background(), &sock.Request{URI: l.base + "%zz" + testListURI, Data: []byte(`{}`)})
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
			}
		})

		t.Run(l.name+" filters are not json", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _ := newResourceErrorHandler(mc)

			res, err := l.fn(h, context.Background(), &sock.Request{URI: l.base + testListURI, Data: []byte(`{not json`)})
			if err != nil || res == nil || res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %v, %v", res, err)
			}
		})

		t.Run(l.name+" filter value has a wrong type", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _ := newResourceErrorHandler(mc)

			res, err := l.fn(h, context.Background(), &sock.Request{URI: l.base + testListURI, Data: []byte(l.bad)})
			if err != nil || res == nil || res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %v, %v", res, err)
			}
		})

		t.Run(l.name+" backend failure", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, m := newResourceErrorHandler(mc)
			l.list(m)

			res, err := l.fn(h, context.Background(), &sock.Request{URI: l.base + testListURI, Data: []byte(`{}`)})
			if err != nil || res == nil || res.StatusCode != 500 {
				t.Errorf("Wrong match. expect: 500, got: %v, %v", res, err)
			}
		})
	}

	t.Run("outbound configs unparseable uri", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newResourceErrorHandler(mc)

		res, err := h.processV1OutboundConfigsGet(context.Background(), &sock.Request{URI: "/v1/outbound_configs%zz" + testListURI})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
		}
	})
}

// Test_resources_backendError verifies how each route reports a failing business handler.
func Test_resources_backendError(t *testing.T) {
	id := uuid.FromStringOrNil(testResID)
	id2 := uuid.FromStringOrNil(testResID2)
	cbURI := "/v1/confbridges/" + testResID
	gcURI := "/v1/groupcalls/" + testResID

	tests := []struct {
		name string

		fn      callsHandlerFunc
		request *sock.Request
		setup   func(m *resourceMocks)

		expectStatus int // checked when expectErr is false
		expectErr    bool
	}{
		{
			name:    "confbridge create",
			fn:      (*listenHandler).processV1ConfbridgesPost,
			request: &sock.Request{URI: "/v1/confbridges", Data: []byte(`{"customer_id":"` + testResID + `","type":"connect"}`)},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().Create(gomock.Any(), id, uuid.Nil, gomock.Any(), uuid.Nil, gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "confbridge terminate",
			fn:      (*listenHandler).processV1ConfbridgesIDTerminatePost,
			request: &sock.Request{URI: cbURI + "/terminate"},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().Terminating(gomock.Any(), id).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "confbridge kick",
			fn:           (*listenHandler).processV1ConfbridgesIDCallsIDDelete,
			request:      &sock.Request{URI: cbURI + "/calls/" + testResID2},
			setup:        func(m *resourceMocks) { m.confbridge.EXPECT().Kick(gomock.Any(), id, id2).Return(errTestBackend) },
			expectStatus: 500,
		},
		{
			name:         "confbridge join",
			fn:           (*listenHandler).processV1ConfbridgesIDCallsIDPost,
			request:      &sock.Request{URI: cbURI + "/calls/" + testResID2},
			setup:        func(m *resourceMocks) { m.confbridge.EXPECT().Join(gomock.Any(), id, id2).Return(errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "confbridge external media start",
			fn:      (*listenHandler).processV1ConfbridgesIDExternalMediaPost,
			request: &sock.Request{URI: cbURI + "/external-media", Data: []byte(`{}`)},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().ExternalMediaStart(gomock.Any(), id, uuid.Nil, "", gomock.Any(), gomock.Any(), "", "", "").Return(nil, errTestBackend)
			},
			expectErr: true,
		},
		{
			name:    "confbridge external media stop",
			fn:      (*listenHandler).processV1ConfbridgesIDExternalMediaDelete,
			request: &sock.Request{URI: cbURI + "/external-media", Data: []byte(`{"external_media_id":"` + testResID2 + `"}`)},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().ExternalMediaStop(gomock.Any(), id, id2).Return(nil, errTestBackend)
			},
			expectErr: true,
		},
		{
			name:    "confbridge recording start",
			fn:      (*listenHandler).processV1ConfbridgesIDRecordingStartPost,
			request: &sock.Request{URI: cbURI + "/recording_start", Data: []byte(`{"duration":30}`)},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().RecordingStart(gomock.Any(), id, gomock.Any(), 0, "", 30, uuid.Nil).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "confbridge recording stop",
			fn:      (*listenHandler).processV1ConfbridgesIDRecordingStopPost,
			request: &sock.Request{URI: cbURI + "/recording_stop"},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().RecordingStop(gomock.Any(), id).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "confbridge flag add",
			fn:      (*listenHandler).processV1ConfbridgesIDFlagsPost,
			request: &sock.Request{URI: cbURI + "/flags", Data: []byte(`{"flag":"no_auto_leave"}`)},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().FlagAdd(gomock.Any(), id, gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "confbridge flag remove",
			fn:      (*listenHandler).processV1ConfbridgesIDFlagsDelete,
			request: &sock.Request{URI: cbURI + "/flags", Data: []byte(`{"flag":"no_auto_leave"}`)},
			setup: func(m *resourceMocks) {
				m.confbridge.EXPECT().FlagRemove(gomock.Any(), id, gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "confbridge ring",
			fn:           (*listenHandler).processV1ConfbridgesIDRingPost,
			request:      &sock.Request{URI: cbURI + "/ring"},
			setup:        func(m *resourceMocks) { m.confbridge.EXPECT().Ring(gomock.Any(), id).Return(errTestBackend) },
			expectStatus: 500,
		},
		{
			name:         "confbridge answer",
			fn:           (*listenHandler).processV1ConfbridgesIDAnswerPost,
			request:      &sock.Request{URI: cbURI + "/answer"},
			setup:        func(m *resourceMocks) { m.confbridge.EXPECT().Answer(gomock.Any(), id).Return(errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "groupcall start",
			fn:      (*listenHandler).processV1GroupcallsPost,
			request: &sock.Request{URI: "/v1/groupcalls", Data: []byte(`{"customer_id":"` + testResID + `"}`)},
			setup: func(m *resourceMocks) {
				m.groupcall.EXPECT().Start(gomock.Any(), gomock.Any(), id, uuid.Nil, gomock.Any(), gomock.Any(), uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "groupcall hangup",
			fn:           (*listenHandler).processV1GroupcallsIDHangupPost,
			request:      &sock.Request{URI: gcURI + "/hangup"},
			setup:        func(m *resourceMocks) { m.groupcall.EXPECT().Hangingup(gomock.Any(), id).Return(nil, errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "groupcall answer",
			fn:      (*listenHandler).processV1GroupcallsIDAnswerGroupcallIDPost,
			request: &sock.Request{URI: gcURI + "/answer_groupcall_id", Data: []byte(`{"answer_groupcall_id":"` + testResID2 + `"}`)},
			setup: func(m *resourceMocks) {
				m.groupcall.EXPECT().AnswerGroupcall(gomock.Any(), id, id2).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "groupcall hangup groupcall",
			fn:      (*listenHandler).processV1GroupcallsIDHangupGroupcallPost,
			request: &sock.Request{URI: gcURI + "/hangup_groupcall"},
			setup: func(m *resourceMocks) {
				m.groupcall.EXPECT().HangupGroupcall(gomock.Any(), id).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "groupcall hangup call",
			fn:           (*listenHandler).processV1GroupcallsIDHangupCallPost,
			request:      &sock.Request{URI: gcURI + "/hangup_call"},
			setup:        func(m *resourceMocks) { m.groupcall.EXPECT().HangupCall(gomock.Any(), id).Return(nil, errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "recording start",
			fn:      (*listenHandler).processV1RecordingsPost,
			request: &sock.Request{URI: "/v1/recordings", Data: []byte(`{"duration":30}`)},
			setup: func(m *resourceMocks) {
				m.recording.EXPECT().Start(gomock.Any(), uuid.Nil, gomock.Any(), uuid.Nil, gomock.Any(), 0, "", 30, uuid.Nil).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "external media start",
			fn:      (*listenHandler).processV1ExternalMediasPost,
			request: &sock.Request{URI: "/v1/external-medias", Data: []byte(`{}`)},
			setup: func(m *resourceMocks) {
				m.externalMedia.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newResourceErrorHandler(mc)
			tt.setup(m)

			res, err := tt.fn(h, context.Background(), tt.request)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectStatus, res)
			}
		})
	}
}

func Test_processV1ConfbridgesIDExternalMediaDelete_missingExternalMediaID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newResourceErrorHandler(mc)

	res, err := h.processV1ConfbridgesIDExternalMediaDelete(context.Background(), &sock.Request{
		URI:  "/v1/confbridges/" + testResID + "/external-media",
		Data: []byte(`{}`),
	})
	if err != nil || res == nil || res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %v, %v", res, err)
	}
}
