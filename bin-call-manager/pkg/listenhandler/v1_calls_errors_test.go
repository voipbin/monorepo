package listenhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/callhandler"
)

const testCallURIID = "4066ee68-d13c-11ed-b9b3-8fe28137c5ad"

var errTestBackend = fmt.Errorf("backend failure")

type callsHandlerFunc func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)

func newCallsErrorHandler(mc *gomock.Controller) (*listenHandler, *callhandler.MockCallHandler) {
	mockCall := callhandler.NewMockCallHandler(mc)
	return &listenHandler{
		sockHandler: sockhandler.NewMockSockHandler(mc),
		callHandler: mockCall,
	}, mockCall
}

// Test_calls_shortURI verifies that a request URI without the id segment is rejected
// with 400 and never reaches the call handler.
func Test_calls_shortURI(t *testing.T) {
	tests := []struct {
		name string
		fn   callsHandlerFunc
		uri  string
	}{
		{"calls post", (*listenHandler).processV1CallsPost, "/v1"},
		{"call get", (*listenHandler).processV1CallsIDGet, "/v1/calls"},
		{"call post", (*listenHandler).processV1CallsIDPost, "/v1/calls"},
		{"call delete", (*listenHandler).processV1CallsIDDelete, "/v1/calls"},
		{"hangup", (*listenHandler).processV1CallsIDHangupPost, "/v1/calls/" + testCallURIID},
		{"health", (*listenHandler).processV1CallsIDHealthPost, "/v1/calls"},
		{"action timeout", (*listenHandler).processV1CallsIDActionTimeoutPost, "/v1/calls"},
		{"action next", (*listenHandler).processV1CallsIDActionNextPost, "/v1/calls"},
		{"chained call ids post", (*listenHandler).processV1CallsIDChainedCallIDsPost, "/v1/calls"},
		{"chained call ids delete", (*listenHandler).processV1CallsIDChainedCallIDsDelete, "/v1/calls/" + testCallURIID},
		{"external media post", (*listenHandler).processV1CallsIDExternalMediaPost, "/v1/calls"},
		{"external media delete", (*listenHandler).processV1CallsIDExternalMediaDelete, "/v1/calls"},
		{"digits get", (*listenHandler).processV1CallsIDDigitsGet, "/v1/calls"},
		{"digits set", (*listenHandler).processV1CallsIDDigitsSet, "/v1/calls"},
		{"recording id put", (*listenHandler).processV1CallsIDRecordingIDPut, "/v1/calls"},
		{"confbridge id put", (*listenHandler).processV1CallsIDConfbridgeIDPut, "/v1/calls"},
		{"recording start", (*listenHandler).processV1CallsIDRecordingStartPost, "/v1/calls"},
		{"recording stop", (*listenHandler).processV1CallsIDRecordingStopPost, "/v1/calls"},
		{"talk", (*listenHandler).processV1CallsIDTalkPost, "/v1/calls"},
		{"play", (*listenHandler).processV1CallsIDPlayPost, "/v1/calls"},
		{"media stop", (*listenHandler).processV1CallsIDMediaStopPost, "/v1/calls"},
		{"hold on", (*listenHandler).processV1CallsIDHoldPost, "/v1/calls"},
		{"hold off", (*listenHandler).processV1CallsIDHoldDelete, "/v1/calls"},
		{"mute on", (*listenHandler).processV1CallsIDMutePost, "/v1/calls"},
		{"mute off", (*listenHandler).processV1CallsIDMuteDelete, "/v1/calls"},
		{"moh on", (*listenHandler).processV1CallsIDMOHPost, "/v1/calls"},
		{"moh off", (*listenHandler).processV1CallsIDMOHDelete, "/v1/calls"},
		{"silence on", (*listenHandler).processV1CallsIDSilencePost, "/v1/calls"},
		{"silence off", (*listenHandler).processV1CallsIDSilenceDelete, "/v1/calls"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newCallsErrorHandler(mc)

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

// Test_calls_invalidBody verifies the handlers that decode a JSON body report a malformed body.
func Test_calls_invalidBody(t *testing.T) {
	// handlers that return the decode error to the caller
	errTests := []struct {
		name string
		fn   callsHandlerFunc
		uri  string
	}{
		{"health", (*listenHandler).processV1CallsIDHealthPost, "/v1/calls/" + testCallURIID + "/health-check"},
		{"action timeout", (*listenHandler).processV1CallsIDActionTimeoutPost, "/v1/calls/" + testCallURIID + "/action-timeout"},
		{"action next", (*listenHandler).processV1CallsIDActionNextPost, "/v1/calls/" + testCallURIID + "/action-next"},
		{"chained call ids post", (*listenHandler).processV1CallsIDChainedCallIDsPost, "/v1/calls/" + testCallURIID + "/chained-call-ids"},
		{"external media post", (*listenHandler).processV1CallsIDExternalMediaPost, "/v1/calls/" + testCallURIID + "/external-media"},
		{"digits set", (*listenHandler).processV1CallsIDDigitsSet, "/v1/calls/" + testCallURIID + "/digits"},
		{"recording id put", (*listenHandler).processV1CallsIDRecordingIDPut, "/v1/calls/" + testCallURIID + "/recording_id"},
		{"confbridge id put", (*listenHandler).processV1CallsIDConfbridgeIDPut, "/v1/calls/" + testCallURIID + "/confbridge_id"},
		{"recording start", (*listenHandler).processV1CallsIDRecordingStartPost, "/v1/calls/" + testCallURIID + "/recording_start"},
		{"talk", (*listenHandler).processV1CallsIDTalkPost, "/v1/calls/" + testCallURIID + "/talk"},
		{"play", (*listenHandler).processV1CallsIDPlayPost, "/v1/calls/" + testCallURIID + "/play"},
		{"mute on", (*listenHandler).processV1CallsIDMutePost, "/v1/calls/" + testCallURIID + "/mute"},
		{"mute off", (*listenHandler).processV1CallsIDMuteDelete, "/v1/calls/" + testCallURIID + "/mute"},
	}
	for _, tt := range errTests {
		t.Run(tt.name+" returns decode error", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newCallsErrorHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{not json`)})
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %v", res)
			}
		})
	}

	// handlers that answer 400 for a malformed body
	badRequestTests := []struct {
		name string
		fn   callsHandlerFunc
		uri  string
	}{
		{"calls post", (*listenHandler).processV1CallsPost, "/v1/calls"},
		{"call post", (*listenHandler).processV1CallsIDPost, "/v1/calls/" + testCallURIID},
		{"external media delete", (*listenHandler).processV1CallsIDExternalMediaDelete, "/v1/calls/" + testCallURIID + "/external-media"},
	}
	for _, tt := range badRequestTests {
		t.Run(tt.name+" returns 400", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newCallsErrorHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{URI: tt.uri, Data: []byte(`{not json`)})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

func Test_processV1CallsGet_badRequest(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectStatus int
		expectErr    bool
	}{
		{
			name:         "filters are not json",
			request:      &sock.Request{URI: "/v1/calls?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{not json`)},
			expectStatus: 400,
		},
		{
			name:         "filter value has a wrong type",
			request:      &sock.Request{URI: "/v1/calls?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expectStatus: 400,
		},
		{
			name:      "uri is not parseable",
			request:   &sock.Request{URI: "/v1/calls%zz?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newCallsErrorHandler(mc)

			res, err := h.processV1CallsGet(context.Background(), tt.request)
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

// Test_calls_backendError verifies that a failure of the call handler is translated to the
// status (or returned error) each route is documented to use.
func Test_calls_backendError(t *testing.T) {
	id := uuid.FromStringOrNil(testCallURIID)
	other := uuid.FromStringOrNil("5ac3b1a2-d13c-11ed-8f7c-5b5e8b5d2f10")
	uri := "/v1/calls/" + testCallURIID

	tests := []struct {
		name string

		fn      callsHandlerFunc
		request *sock.Request
		setup   func(m *callhandler.MockCallHandler)

		expectStatus int // checked when expectErr is false
		expectErr    bool
	}{
		{
			name:    "list",
			fn:      (*listenHandler).processV1CallsGet,
			request: &sock.Request{URI: "/v1/calls?page_size=10", Data: []byte(`{}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "get",
			fn:           (*listenHandler).processV1CallsIDGet,
			request:      &sock.Request{URI: uri},
			setup:        func(m *callhandler.MockCallHandler) { m.EXPECT().Get(gomock.Any(), id).Return(nil, errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "create calls",
			fn:      (*listenHandler).processV1CallsPost,
			request: &sock.Request{URI: "/v1/calls", Data: []byte(`{"customer_id":"` + other.String() + `"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().CreateCallsOutgoing(gomock.Any(), other, uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), false, false, "", gomock.Any(), gomock.Any()).Return(nil, nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "create call",
			fn:      (*listenHandler).processV1CallsIDPost,
			request: &sock.Request{URI: uri, Data: []byte(`{"customer_id":"` + other.String() + `"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().CreateCallOutgoing(gomock.Any(), id, other, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), false, false, "", gomock.Any(), gomock.Any()).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "delete",
			fn:           (*listenHandler).processV1CallsIDDelete,
			request:      &sock.Request{URI: uri},
			setup:        func(m *callhandler.MockCallHandler) { m.EXPECT().Delete(gomock.Any(), id).Return(nil, errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "hangup",
			fn:      (*listenHandler).processV1CallsIDHangupPost,
			request: &sock.Request{URI: uri + "/hangup"},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().HangingUp(gomock.Any(), id, call.HangupReasonNormal).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "action timeout",
			fn:      (*listenHandler).processV1CallsIDActionTimeoutPost,
			request: &sock.Request{URI: uri + "/action-timeout", Data: []byte(`{}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().ActionTimeout(gomock.Any(), id, gomock.Any()).Return(errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:         "action next get",
			fn:           (*listenHandler).processV1CallsIDActionNextPost,
			request:      &sock.Request{URI: uri + "/action-next", Data: []byte(`{}`)},
			setup:        func(m *callhandler.MockCallHandler) { m.EXPECT().Get(gomock.Any(), id).Return(nil, errTestBackend) },
			expectStatus: 500,
		},
		{
			name:    "chained call add",
			fn:      (*listenHandler).processV1CallsIDChainedCallIDsPost,
			request: &sock.Request{URI: uri + "/chained-call-ids", Data: []byte(`{"chained_call_id":"` + other.String() + `"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().ChainedCallIDAdd(gomock.Any(), id, other).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "chained call remove",
			fn:      (*listenHandler).processV1CallsIDChainedCallIDsDelete,
			request: &sock.Request{URI: uri + "/chained-call-ids/" + other.String()},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().ChainedCallIDRemove(gomock.Any(), id, other).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "external media start",
			fn:      (*listenHandler).processV1CallsIDExternalMediaPost,
			request: &sock.Request{URI: uri + "/external-media", Data: []byte(`{}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().ExternalMediaStart(gomock.Any(), id, uuid.Nil, "", gomock.Any(), gomock.Any(), "", "", "", gomock.Any(), gomock.Any()).Return(nil, errTestBackend)
			},
			expectErr: true,
		},
		{
			name:    "external media stop",
			fn:      (*listenHandler).processV1CallsIDExternalMediaDelete,
			request: &sock.Request{URI: uri + "/external-media", Data: []byte(`{"external_media_id":"` + other.String() + `"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().ExternalMediaStop(gomock.Any(), id, other).Return(nil, errTestBackend)
			},
			expectErr: true,
		},
		{
			name:    "digits get",
			fn:      (*listenHandler).processV1CallsIDDigitsGet,
			request: &sock.Request{URI: uri + "/digits"},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().DigitsGet(gomock.Any(), id).Return("", errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "digits set",
			fn:      (*listenHandler).processV1CallsIDDigitsSet,
			request: &sock.Request{URI: uri + "/digits", Data: []byte(`{"digits":"12"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().DigitsSet(gomock.Any(), id, "12").Return(errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "recording id update",
			fn:      (*listenHandler).processV1CallsIDRecordingIDPut,
			request: &sock.Request{URI: uri + "/recording_id", Data: []byte(`{"recording_id":"` + other.String() + `"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().UpdateRecordingID(gomock.Any(), id, other).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "confbridge id update",
			fn:      (*listenHandler).processV1CallsIDConfbridgeIDPut,
			request: &sock.Request{URI: uri + "/confbridge_id", Data: []byte(`{"confbridge_id":"` + other.String() + `"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().UpdateConfbridgeID(gomock.Any(), id, other).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "recording start",
			fn:      (*listenHandler).processV1CallsIDRecordingStartPost,
			request: &sock.Request{URI: uri + "/recording_start", Data: []byte(`{"format":"wav","duration":30}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().RecordingStart(gomock.Any(), id, gomock.Any(), 0, "", 30, uuid.Nil).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "recording stop",
			fn:      (*listenHandler).processV1CallsIDRecordingStopPost,
			request: &sock.Request{URI: uri + "/recording_stop"},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().RecordingStop(gomock.Any(), id).Return(nil, errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "talk",
			fn:      (*listenHandler).processV1CallsIDTalkPost,
			request: &sock.Request{URI: uri + "/talk", Data: []byte(`{"text":"hello","language":"en-US"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().Talk(gomock.Any(), id, false, "hello", "en-US", "", "").Return(errTestBackend)
			},
			expectStatus: 500,
		},
		{
			name:    "play",
			fn:      (*listenHandler).processV1CallsIDPlayPost,
			request: &sock.Request{URI: uri + "/play", Data: []byte(`{"media_urls":["https://example.invalid/a.wav"]}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().Play(gomock.Any(), id, false, []string{"https://example.invalid/a.wav"}).Return(errTestBackend)
			},
			expectErr: true,
		},
		{
			name:         "media stop",
			fn:           (*listenHandler).processV1CallsIDMediaStopPost,
			request:      &sock.Request{URI: uri + "/media_stop"},
			setup:        func(m *callhandler.MockCallHandler) { m.EXPECT().MediaStop(gomock.Any(), id).Return(errTestBackend) },
			expectStatus: 500,
		},
		{
			name:      "hold on",
			fn:        (*listenHandler).processV1CallsIDHoldPost,
			request:   &sock.Request{URI: uri + "/hold"},
			setup:     func(m *callhandler.MockCallHandler) { m.EXPECT().HoldOn(gomock.Any(), id).Return(errTestBackend) },
			expectErr: true,
		},
		{
			name:      "hold off",
			fn:        (*listenHandler).processV1CallsIDHoldDelete,
			request:   &sock.Request{URI: uri + "/hold"},
			setup:     func(m *callhandler.MockCallHandler) { m.EXPECT().HoldOff(gomock.Any(), id).Return(errTestBackend) },
			expectErr: true,
		},
		{
			name:    "mute on",
			fn:      (*listenHandler).processV1CallsIDMutePost,
			request: &sock.Request{URI: uri + "/mute", Data: []byte(`{"direction":"in"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().MuteOn(gomock.Any(), id, call.MuteDirectionIn).Return(errTestBackend)
			},
			expectErr: true,
		},
		{
			name:    "mute off",
			fn:      (*listenHandler).processV1CallsIDMuteDelete,
			request: &sock.Request{URI: uri + "/mute", Data: []byte(`{"direction":"out"}`)},
			setup: func(m *callhandler.MockCallHandler) {
				m.EXPECT().MuteOff(gomock.Any(), id, call.MuteDirectionOut).Return(errTestBackend)
			},
			expectErr: true,
		},
		{
			name:      "moh on",
			fn:        (*listenHandler).processV1CallsIDMOHPost,
			request:   &sock.Request{URI: uri + "/moh"},
			setup:     func(m *callhandler.MockCallHandler) { m.EXPECT().MOHOn(gomock.Any(), id).Return(errTestBackend) },
			expectErr: true,
		},
		{
			name:      "moh off",
			fn:        (*listenHandler).processV1CallsIDMOHDelete,
			request:   &sock.Request{URI: uri + "/moh"},
			setup:     func(m *callhandler.MockCallHandler) { m.EXPECT().MOHOff(gomock.Any(), id).Return(errTestBackend) },
			expectErr: true,
		},
		{
			name:      "silence on",
			fn:        (*listenHandler).processV1CallsIDSilencePost,
			request:   &sock.Request{URI: uri + "/silence"},
			setup:     func(m *callhandler.MockCallHandler) { m.EXPECT().SilenceOn(gomock.Any(), id).Return(errTestBackend) },
			expectErr: true,
		},
		{
			name:      "silence off",
			fn:        (*listenHandler).processV1CallsIDSilenceDelete,
			request:   &sock.Request{URI: uri + "/silence"},
			setup:     func(m *callhandler.MockCallHandler) { m.EXPECT().SilenceOff(gomock.Any(), id).Return(errTestBackend) },
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockCall := newCallsErrorHandler(mc)
			tt.setup(mockCall)

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

// Test_processV1CallsIDActionTimeoutPost_tmExecute verifies how the tm_execute value is converted.
func Test_processV1CallsIDActionTimeoutPost_tmExecute(t *testing.T) {
	id := uuid.FromStringOrNil(testCallURIID)
	actionID := uuid.FromStringOrNil("5ac3b1a2-d13c-11ed-8f7c-5b5e8b5d2f10")

	tests := []struct {
		name string

		tmExecute string

		expectTM *time.Time
	}{
		{
			name:      "rfc3339 value",
			tmExecute: "2024-03-04T05:06:07.123456789Z",
			expectTM:  timePtr(time.Date(2024, 3, 4, 5, 6, 7, 123456789, time.UTC)),
		},
		{
			name:      "legacy layout value",
			tmExecute: "2024-03-04 05:06:07.123456",
			expectTM:  timePtr(time.Date(2024, 3, 4, 5, 6, 7, 123456000, time.UTC)),
		},
		{
			name:      "unparseable value is dropped",
			tmExecute: "yesterday",
			expectTM:  nil,
		},
		{
			name:      "empty value",
			tmExecute: "",
			expectTM:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockCall := newCallsErrorHandler(mc)

			mockCall.EXPECT().ActionTimeout(gomock.Any(), id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, a *fmaction.Action) error {
				if a.ID != actionID {
					t.Errorf("Wrong action id. expect: %s, got: %s", actionID, a.ID)
				}
				switch {
				case tt.expectTM == nil && a.TMExecute != nil:
					t.Errorf("Wrong tm_execute. expect: nil, got: %v", a.TMExecute)
				case tt.expectTM != nil && (a.TMExecute == nil || !a.TMExecute.Equal(*tt.expectTM)):
					t.Errorf("Wrong tm_execute. expect: %v, got: %v", tt.expectTM, a.TMExecute)
				}
				return nil
			})

			body := `{"action_id":"` + actionID.String() + `","tm_execute":"` + tt.tmExecute + `"}`
			res, err := h.processV1CallsIDActionTimeoutPost(context.Background(), &sock.Request{URI: "/v1/calls/" + testCallURIID + "/action-timeout", Data: []byte(body)})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %v", res)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func Test_processV1CallsIDExternalMediaDelete_missingExternalMediaID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newCallsErrorHandler(mc)

	res, err := h.processV1CallsIDExternalMediaDelete(context.Background(), &sock.Request{
		URI:  "/v1/calls/" + testCallURIID + "/external-media",
		Data: []byte(`{}`),
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.StatusCode != 400 {
		t.Errorf("Wrong match. expect: 400, got: %v", res)
	}
}
