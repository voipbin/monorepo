package listenhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/builderhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// secretInput is a user-input marker. It must reach no log line, whatever
// fails: the body is the customer's own business description.
const secretInput = "SECRET-INPUT-STRING-do-not-log"

var builderCustomerID = uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

func builderChatRequest(t *testing.T, customerID uuid.UUID, content string) *sock.Request {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"customer_id": customerID,
		"messages":    []map[string]string{{"role": "user", "content": content}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &sock.Request{URI: builder.URIChat, Method: sock.RequestMethodPost, DataType: "application/json", Data: data}
}

func newBuilderListenHandler(t *testing.T) (*listenHandler, *builderhandler.MockBuilderHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	bh := builderhandler.NewMockBuilderHandler(mc)
	return &listenHandler{builderHandler: bh}, bh
}

func Test_isBuilderRoute(t *testing.T) {
	tests := []struct {
		uri    string
		method sock.RequestMethod
		want   bool
	}{
		{builder.URIChat, sock.RequestMethodPost, true},
		{builder.URIStatus, sock.RequestMethodGet, true},
		// The route is claimed by URI alone, whatever the method, so a wrong
		// method is answered here and never falls through to the default 404
		// branch whose log line carries the whole request.
		{builder.URIChat, sock.RequestMethodGet, true},
		{builder.URIStatus, sock.RequestMethodPost, true},
		{builder.URIChat, sock.RequestMethodDelete, true},
		// Exact match only: a longer URI or a query string is not ours.
		{builder.URIChat + "/", sock.RequestMethodPost, false},
		{builder.URIChat + "?x=1", sock.RequestMethodPost, false},
		{"/v1/ai_builder/chats", sock.RequestMethodPost, false},
		{"/v1/ais", sock.RequestMethodGet, false},
		{"", sock.RequestMethodGet, false},
	}
	for _, tt := range tests {
		t.Run(tt.uri+"/"+string(tt.method), func(t *testing.T) {
			if got := isBuilderRoute(&sock.Request{URI: tt.uri, Method: tt.method}); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processBuilder_chatSuccess(t *testing.T) {
	h, bh := newBuilderListenHandler(t)
	bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, id uuid.UUID, req *builder.ChatRequest) (*builder.ChatResponse, error) {
			if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
				t.Errorf("the request must reach the handler intact: %+v", req)
			}
			return &builder.ChatResponse{Message: "hi"}, nil
		})

	resp, err := h.processRequest(builderChatRequest(t, builderCustomerID, "hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var out builder.ChatResponse
	if err := json.Unmarshal(resp.Data, &out); err != nil || out.Message != "hi" {
		t.Errorf("response body: %s (%v)", resp.Data, err)
	}
}

// The trusted customer id is the one api-manager put in the RPC DTO. A request
// without it must be refused, not handled for the nil customer.
func Test_processBuilder_missingCustomerIDIsRejected(t *testing.T) {
	h, _ := newBuilderListenHandler(t) // no Chat call expected
	resp, err := h.processRequest(builderChatRequest(t, uuid.Nil, "hello"))
	if err != nil {
		t.Fatalf("a handler error must be an (response, nil) pair, got err: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: %d", resp.StatusCode)
	}
}

func Test_processBuilder_status(t *testing.T) {
	for _, available := range []bool{true, false} {
		h, bh := newBuilderListenHandler(t)
		bh.EXPECT().Status().Return(&builder.StatusResponse{Available: available, MaxMessages: 40, MaxMessageChars: 2000})

		resp, err := h.processRequest(&sock.Request{URI: builder.URIStatus, Method: sock.RequestMethodGet})
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("status route: %v / %+v", err, resp)
		}
		var out builder.StatusResponse
		if err := json.Unmarshal(resp.Data, &out); err != nil || out.Available != available {
			t.Errorf("available=%v: got %s (%v)", available, resp.Data, err)
		}
	}
}

// Every failure is turned into a typed response here. Returning the error would
// reach the queue consumer (consume.go), which logs err and publishes a bare
// 500, losing the reason the client needs.
func Test_processBuilder_everyErrorBecomesAResponse(t *testing.T) {
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
		{"unavailable", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "x"), http.StatusServiceUnavailable, builder.ReasonUnavailable},
		{"disabled", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonDisabled, "x"), http.StatusServiceUnavailable, builder.ReasonDisabled},
		{"invalid argument", cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "x"), http.StatusBadRequest, builder.ReasonInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, bh := newBuilderListenHandler(t)
			bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).Return(nil, tt.chatErr)

			resp, err := h.processRequest(builderChatRequest(t, builderCustomerID, "hello"))
			if err != nil {
				t.Fatalf("the error must become a response, not be returned: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			// What api-manager sees after the wire: the same reason.
			ve := cerrors.FromResponse(resp)
			if ve == nil || ve.Reason != tt.wantReason {
				t.Errorf("the reason must survive the wire: got %+v, want %s", ve, tt.wantReason)
			}
		})
	}

	t.Run("an untyped error becomes a bare 500", func(t *testing.T) {
		h, bh := newBuilderListenHandler(t)
		bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).Return(nil, errors.New("plain"))
		resp, err := h.processRequest(builderChatRequest(t, builderCustomerID, "hello"))
		if err != nil || resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("got %v / %+v", err, resp)
		}
	})
}

func Test_processBuilder_badBodyIsA400WithoutEchoingIt(t *testing.T) {
	for name, body := range map[string][]byte{
		"not json":     []byte("{" + secretInput),
		"wrong type":   []byte(`{"customer_id":"` + builderCustomerID.String() + `","messages":"` + secretInput + `"}`),
		"empty":        nil,
		"bad customer": []byte(`{"customer_id":"` + secretInput + `","messages":[]}`),
	} {
		t.Run(name, func(t *testing.T) {
			h, _ := newBuilderListenHandler(t) // no Chat call expected
			resp, err := h.processRequest(&sock.Request{URI: builder.URIChat, Method: sock.RequestMethodPost, DataType: "application/json", Data: body})
			if err != nil {
				t.Fatalf("got err: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status: %d", resp.StatusCode)
			}
			if strings.Contains(string(resp.Data), secretInput) {
				t.Errorf("the response body echoes the input: %s", resp.Data)
			}
		})
	}
}

func Test_processBuilder_wrongMethodIs400(t *testing.T) {
	h, _ := newBuilderListenHandler(t)
	for _, m := range []sock.RequestMethod{sock.RequestMethodGet, sock.RequestMethodPut, sock.RequestMethodDelete} {
		resp, err := h.processRequest(&sock.Request{URI: builder.URIChat, Method: m})
		if err != nil || resp.StatusCode != http.StatusBadRequest {
			t.Errorf("chat with %s: %v / %+v", m, err, resp)
		}
	}
	resp, err := h.processRequest(&sock.Request{URI: builder.URIStatus, Method: sock.RequestMethodPost})
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status with POST: %v / %+v", err, resp)
	}
}

// A panic anywhere below becomes a bare 500: no body, no panic value (it can
// carry the input), and no crash of the RPC worker.
func Test_processBuilder_panicBecomesA500WithoutTheInput(t *testing.T) {
	var buf bytes.Buffer
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	h, bh := newBuilderListenHandler(t)
	bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, _ uuid.UUID, req *builder.ChatRequest) (*builder.ChatResponse, error) {
			panic("boom with " + req.Messages[0].Content)
		})

	resp, err := h.processRequest(builderChatRequest(t, builderCustomerID, secretInput))
	if err != nil {
		t.Fatalf("a panic must become a response, got err: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status: %d", resp.StatusCode)
	}
	if strings.Contains(buf.String(), secretInput) || strings.Contains(string(resp.Data), secretInput) {
		t.Errorf("the panic value (which carried the input) leaked:\nlog: %s\nbody: %s", buf.String(), resp.Data)
	}
}

// The whole path through processRequest, for every way a turn can fail: no log
// line from this package, from the handler it calls, or from the surrounding
// request logging may carry the input. This is the test that the "route before
// the log is built" structure exists to pass: processRequest's own log line
// puts the whole request into a field.
func Test_processBuilder_neverLogsTheInput(t *testing.T) {
	var buf bytes.Buffer
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	run := func(name string, req *sock.Request, setup func(bh *builderhandler.MockBuilderHandler)) {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			h, bh := newBuilderListenHandler(t)
			if setup != nil {
				setup(bh)
			}
			resp, err := h.processRequest(req)
			if err != nil {
				t.Fatalf("got err: %v", err)
			}
			if strings.Contains(buf.String(), secretInput) {
				t.Errorf("a log line carries the input:\n%s", buf.String())
			}
			if resp != nil && strings.Contains(string(resp.Data), secretInput) {
				t.Errorf("the response carries the input: %s", resp.Data)
			}
		})
	}

	good := builderChatRequest(t, builderCustomerID, secretInput)
	run("success", good, func(bh *builderhandler.MockBuilderHandler) {
		bh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(&builder.ChatResponse{Message: "ok"}, nil)
	})
	run("llm error", good, func(bh *builderhandler.MockBuilderHandler) {
		bh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "x"))
	})
	run("untyped error echoing the input", good, func(bh *builderhandler.MockBuilderHandler) {
		bh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("provider said: "+secretInput))
	})
	run("typed error with the input as its cause", good, func(bh *builderhandler.MockBuilderHandler) {
		bh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil,
			cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "x").Wrap(errors.New(secretInput)))
	})
	run("unmarshal failure", &sock.Request{URI: builder.URIChat, Method: sock.RequestMethodPost, Data: []byte("{" + secretInput)}, nil)
	run("wrong method with a body", &sock.Request{URI: builder.URIChat, Method: sock.RequestMethodGet, Data: []byte(secretInput)}, nil)
	run("panic", good, func(bh *builderhandler.MockBuilderHandler) {
		bh.EXPECT().Chat(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, _ uuid.UUID, req *builder.ChatRequest) (*builder.ChatResponse, error) {
				panic(req.Messages[0].Content)
			})
	})
}

// A request that is not a Builder route still goes through the existing code
// path unchanged.
func Test_processRequest_otherRoutesAreUntouched(t *testing.T) {
	h, _ := newBuilderListenHandler(t)
	resp, err := h.processRequest(&sock.Request{URI: "/v1/no-such-route", Method: sock.RequestMethodGet})
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Errorf("got %v / %+v", err, resp)
	}
}

// A wrong method must be refused even when its body is a perfectly valid chat
// request: otherwise a GET with a body would run (and be charged as) a turn.
func Test_processBuilder_wrongMethodWithAValidBodyIsNeverRun(t *testing.T) {
	h, _ := newBuilderListenHandler(t) // no Chat call expected
	good := builderChatRequest(t, builderCustomerID, "hello")
	for _, m := range []sock.RequestMethod{sock.RequestMethodGet, sock.RequestMethodPut, sock.RequestMethodDelete} {
		req := &sock.Request{URI: good.URI, Method: m, DataType: good.DataType, Data: good.Data}
		resp, err := h.processRequest(req)
		if err != nil || resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s with a valid body: %v / %+v", m, err, resp)
		}
	}
}

// The draft the client sends is passed to the model, or the "refine the form"
// turn would silently start from nothing.
func Test_processBuilder_chatPassesTheCurrentDraftOn(t *testing.T) {
	h, bh := newBuilderListenHandler(t)
	bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, _ uuid.UUID, req *builder.ChatRequest) (*builder.ChatResponse, error) {
			if req.CurrentDraft == nil || req.CurrentDraft.Name != "Clinic desk" {
				t.Errorf("the current draft was not passed on: %+v", req.CurrentDraft)
			}
			return &builder.ChatResponse{Message: "ok"}, nil
		})

	data, _ := json.Marshal(map[string]any{
		"customer_id":   builderCustomerID,
		"messages":      []map[string]string{{"role": "user", "content": "hi"}},
		"current_draft": map[string]any{"name": "Clinic desk"},
	})
	resp, err := h.processRequest(&sock.Request{URI: builder.URIChat, Method: sock.RequestMethodPost, DataType: "application/json", Data: data})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("got %v / %+v", err, resp)
	}
}

var updateBuilderGolden = flag.Bool("update", false, "rewrite the builder wire golden files in testdata")

// builderWireReasons are the reasons whose wire form api-manager must be able
// to restore (design 4.7). The golden file of each is what ai-manager really
// puts on the queue for that failure, taken from processRequest's output, not
// from a hand-built value.
//
// api-manager keeps a copy of each file under
// bin-api-manager/pkg/servicehandler/testdata and restores it with
// cerrors.FromResponse. The two modules cannot import each other's tests, so
// the copies are kept equal by a person reading the diff: the checklist in
// docs/builder-implementation-checks.md says so. Change a file here, change
// its copy there.
var builderWireReasons = []struct {
	file string
	err  error
}{
	{"builder_busy.json", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonBusy, "the assistant builder is busy, try again shortly")},
	{"builder_daily_limit.json", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "the daily limit of the assistant builder has been reached")},
	{"builder_timeout.json", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "the assistant builder took too long, try again")},
	{"builder_response_invalid.json", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "the assistant builder gave an unusable answer, try again")},
	{"builder_unavailable.json", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "the assistant builder is not available")},
}

func Test_processBuilder_wireFormatMatchesTheGoldenFiles(t *testing.T) {
	for _, tt := range builderWireReasons {
		t.Run(tt.file, func(t *testing.T) {
			h, bh := newBuilderListenHandler(t)
			// The cause is attached on purpose: it must not reach the wire.
			failure := tt.err.(*cerrors.VoipbinError).Wrap(errors.New(secretInput))
			bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).Return(nil, failure)

			resp, err := h.processRequest(builderChatRequest(t, builderCustomerID, "hello"))
			if err != nil {
				t.Fatalf("got err: %v", err)
			}
			got, errMarshal := json.MarshalIndent(resp, "", "  ")
			if errMarshal != nil {
				t.Fatal(errMarshal)
			}
			got = append(got, '\n')

			path := filepath.Join("testdata", tt.file)
			if *updateBuilderGolden {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, errRead := os.ReadFile(path)
			if errRead != nil {
				t.Fatalf("cannot read the golden file (run with -update once): %v", errRead)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("the wire form changed.\nwant: %s\ngot:  %s", want, got)
			}
			if strings.Contains(string(got), secretInput) {
				t.Error("the error cause reached the wire")
			}
		})
	}
}

// A recovered panic is counted under its own fixed label.
func Test_processBuilder_panicIsCounted(t *testing.T) {
	before := panicCount(t)
	h, bh := newBuilderListenHandler(t)
	bh.EXPECT().Chat(gomock.Any(), builderCustomerID, gomock.Any()).DoAndReturn(
		func(_ any, _ uuid.UUID, _ *builder.ChatRequest) (*builder.ChatResponse, error) { panic("boom") })

	resp, err := h.processRequest(builderChatRequest(t, builderCustomerID, "hello"))
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("got %v / %+v", err, resp)
	}
	if panicCount(t)-before != 1 {
		t.Error("the panic must be counted")
	}
}

// panicCount reads ai_manager_builder_chat_total{result="internal"} from the
// default registry, because the counter itself is private to builderhandler.
func panicCount(t *testing.T) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() != "ai_manager_builder_chat_total" {
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
