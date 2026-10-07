package listenhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

func flowChatRequest(t *testing.T, customerID uuid.UUID, content string) *sock.Request {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"customer_id":            customerID,
		"messages":               []map[string]string{{"role": "user", "content": content}},
		"supported_action_types": []string{"talk", "hangup"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &sock.Request{URI: flowbuilder.URIChat, Method: sock.RequestMethodPost, DataType: "application/json", Data: data}
}

func newFlowBuilderListenHandler(t *testing.T) (*listenHandler, *builderhandler.MockFlowBuilderHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	fh := builderhandler.NewMockFlowBuilderHandler(mc)
	return &listenHandler{flowBuilderHandler: fh}, fh
}

func Test_isBuilderRoute_flowBuilder(t *testing.T) {
	tests := []struct {
		uri    string
		method sock.RequestMethod
		want   bool
	}{
		{flowbuilder.URIChat, sock.RequestMethodPost, true},
		// Claimed by URI alone, so a wrong method never falls through to the
		// default 404 branch whose log line carries the whole request.
		{flowbuilder.URIChat, sock.RequestMethodGet, true},
		{flowbuilder.URIChat + "/", sock.RequestMethodPost, false},
		{flowbuilder.URIChat + "?x=1", sock.RequestMethodPost, false},
		{"/v1/flow_builder/chats", sock.RequestMethodPost, false},
		{"/v1/flow_builder/status", sock.RequestMethodGet, false},
	}
	for _, tt := range tests {
		t.Run(tt.uri+"/"+string(tt.method), func(t *testing.T) {
			if got := isBuilderRoute(&sock.Request{URI: tt.uri, Method: tt.method}); got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}
}

func Test_processFlowBuilder_chatSuccess(t *testing.T) {
	h, fh := newFlowBuilderListenHandler(t)
	fh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, _ uuid.UUID, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
			if len(req.Messages) != 1 || req.Messages[0].Content != "hello" || len(req.SupportedActionTypes) != 2 {
				t.Errorf("Wrong match. the request must reach the handler intact: %+v", req)
			}
			return &flowbuilder.ChatResponse{Message: "hi"}, nil
		})

	resp, err := h.processRequest(flowChatRequest(t, builderCustomerID, "hello"))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. expect: 200, got: %v / %+v", err, resp)
	}
	var out flowbuilder.ChatResponse
	if err := json.Unmarshal(resp.Data, &out); err != nil || out.Message != "hi" {
		t.Errorf("Wrong match. response body: %s (%v)", resp.Data, err)
	}
}

func Test_processFlowBuilder_passesTheCurrentDraftOn(t *testing.T) {
	h, fh := newFlowBuilderListenHandler(t)
	fh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, _ uuid.UUID, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
			if req.CurrentDraft == nil || len(req.CurrentDraft.Actions) != 1 || req.CurrentDraft.Labels["a"] != "start" {
				t.Errorf("Wrong match. the draft was not passed on: %+v", req.CurrentDraft)
			}
			return &flowbuilder.ChatResponse{Message: "ok"}, nil
		})

	data, _ := json.Marshal(map[string]any{
		"customer_id":            builderCustomerID,
		"messages":               []map[string]string{{"role": "user", "content": "hi"}},
		"supported_action_types": []string{"talk"},
		"current_draft": map[string]any{
			"actions": []map[string]any{{"id": "a", "type": "talk"}},
			"labels":  map[string]string{"a": "start"},
		},
	})
	resp, err := h.processRequest(&sock.Request{URI: flowbuilder.URIChat, Method: sock.RequestMethodPost, DataType: "application/json", Data: data})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. got %v / %+v", err, resp)
	}
}

func Test_processFlowBuilder_missingCustomerIDIsRejected(t *testing.T) {
	h, _ := newFlowBuilderListenHandler(t) // no Chat call expected
	resp, err := h.processRequest(flowChatRequest(t, uuid.Nil, "hello"))
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: 400, got: %v / %+v", err, resp)
	}
}

func Test_processFlowBuilder_everyErrorBecomesAResponse(t *testing.T) {
	tests := []struct {
		name       string
		chatErr    error
		wantStatus int
		wantReason string
	}{
		{"daily limit", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "x"), http.StatusTooManyRequests, builder.ReasonDailyLimit},
		{"busy", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonBusy, "x"), http.StatusTooManyRequests, builder.ReasonBusy},
		{"timeout", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x"), http.StatusServiceUnavailable, builder.ReasonTimeout},
		{"response invalid", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "x"), http.StatusServiceUnavailable, builder.ReasonResponseInvalid},
		{"invalid argument", cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "x"), http.StatusBadRequest, builder.ReasonInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, fh := newFlowBuilderListenHandler(t)
			fh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).Return(nil, tt.chatErr)

			resp, err := h.processRequest(flowChatRequest(t, builderCustomerID, "hello"))
			if err != nil {
				t.Fatalf("Wrong match. the error must become a response: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.wantStatus, resp.StatusCode)
			}
			if ve := cerrors.FromResponse(resp); ve == nil || ve.Reason != tt.wantReason {
				t.Errorf("Wrong match. the reason must survive the wire: got %+v, want %s", ve, tt.wantReason)
			}
		})
	}
}

func Test_processFlowBuilder_wrongMethodIsNeverRun(t *testing.T) {
	h, _ := newFlowBuilderListenHandler(t) // no Chat call expected
	good := flowChatRequest(t, builderCustomerID, "hello")
	for _, m := range []sock.RequestMethod{sock.RequestMethodGet, sock.RequestMethodPut, sock.RequestMethodDelete} {
		resp, err := h.processRequest(&sock.Request{URI: good.URI, Method: m, DataType: good.DataType, Data: good.Data})
		if err != nil || resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Wrong match. %s with a valid body: %v / %+v", m, err, resp)
		}
	}
}

func Test_processFlowBuilder_badBodyIsA400WithoutEchoingIt(t *testing.T) {
	for name, body := range map[string][]byte{
		"not json":   []byte("{" + secretInput),
		"wrong type": []byte(`{"customer_id":"` + builderCustomerID.String() + `","messages":"` + secretInput + `"}`),
		"empty":      nil,
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := newFlowBuilderListenHandler(t) // no Chat call expected
			resp, err := h.processRequest(&sock.Request{URI: flowbuilder.URIChat, Method: sock.RequestMethodPost, DataType: "application/json", Data: body})
			if err != nil || resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("Wrong match. expect: 400, got: %v / %+v", err, resp)
			}
			if strings.Contains(string(resp.Data), secretInput) {
				t.Errorf("Wrong match. the response body echoes the input: %s", resp.Data)
			}
		})
	}
}

// A panic becomes a bare 500 that carries no input, and it is counted in the
// Flow series only.
func Test_processFlowBuilder_panicIsCountedInTheFlowSeriesOnly(t *testing.T) {
	var buf bytes.Buffer
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	flowBefore := internalCount(t, "ai_manager_flow_builder_chat_total")
	assistantBefore := internalCount(t, "ai_manager_builder_chat_total")

	h, fh := newFlowBuilderListenHandler(t)
	fh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, _ uuid.UUID, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
			panic("boom with " + req.Messages[0].Content)
		})

	resp, err := h.processRequest(flowChatRequest(t, builderCustomerID, secretInput))
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Wrong match. expect: bare 500, got: %v / %+v", err, resp)
	}
	if strings.Contains(buf.String(), secretInput) || strings.Contains(string(resp.Data), secretInput) {
		t.Errorf("Wrong match. the panic value leaked:\nlog: %s\nbody: %s", buf.String(), resp.Data)
	}
	if got := internalCount(t, "ai_manager_flow_builder_chat_total") - flowBefore; got != 1 {
		t.Errorf("Wrong match. expect: flow +1, got: %v", got)
	}
	if got := internalCount(t, "ai_manager_builder_chat_total") - assistantBefore; got != 0 {
		t.Errorf("Wrong match. expect: assistant untouched, got: %v", got)
	}
}

// The whole path through processRequest, for every way a turn can fail: no log
// line from this package, from the handler it calls, or from the surrounding
// request logging may carry the input.
func Test_processFlowBuilder_neverLogsTheInput(t *testing.T) {
	var buf bytes.Buffer
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	run := func(name string, req *sock.Request, setup func(fh *builderhandler.MockFlowBuilderHandler)) {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			h, fh := newFlowBuilderListenHandler(t)
			if setup != nil {
				setup(fh)
			}
			resp, err := h.processRequest(req)
			if err != nil {
				t.Fatalf("Wrong match. got err: %v", err)
			}
			if strings.Contains(buf.String(), secretInput) {
				t.Errorf("Wrong match. a log line carries the input:\n%s", buf.String())
			}
			if resp != nil && strings.Contains(string(resp.Data), secretInput) {
				t.Errorf("Wrong match. the response carries the input: %s", resp.Data)
			}
		})
	}

	good := flowChatRequest(t, builderCustomerID, secretInput)
	run("success", good, func(fh *builderhandler.MockFlowBuilderHandler) {
		fh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(&flowbuilder.ChatResponse{Message: "ok"}, nil)
	})
	run("untyped error echoing the input", good, func(fh *builderhandler.MockFlowBuilderHandler) {
		fh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("provider said: "+secretInput))
	})
	run("typed error with the input as its cause", good, func(fh *builderhandler.MockFlowBuilderHandler) {
		fh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil,
			cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "x").Wrap(errors.New(secretInput)))
	})
	run("unmarshal failure", &sock.Request{URI: flowbuilder.URIChat, Method: sock.RequestMethodPost, Data: []byte("{" + secretInput)}, nil)
	run("wrong method with a body", &sock.Request{URI: flowbuilder.URIChat, Method: sock.RequestMethodGet, Data: []byte(secretInput)}, nil)
	run("panic", good, func(fh *builderhandler.MockFlowBuilderHandler) {
		fh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, _ uuid.UUID, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error) {
				panic(req.Messages[0].Content)
			})
	})
}

// panicCount reads the value of the result="internal" series of a builder
// counter from the default registry, which is where the handlers register.
func internalCount(t *testing.T, name string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Wrong match. could not gather metrics: %v", err)
	}
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "result" && l.GetValue() == "internal" {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}
