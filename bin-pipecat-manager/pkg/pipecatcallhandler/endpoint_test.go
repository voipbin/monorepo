package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"
)

const (
	testEndpointURL = "https://llm.example-endpoint.test/v1"
	testEndpointKey = "dummy-endpoint-key-7f3a"
)

var testEndpointCustomerID = uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")

// stubEndpointResolver replaces the name resolver of the ai model package for one test.
func stubEndpointResolver(t *testing.T, addrs ...string) {
	t.Helper()

	parsed := []netip.Addr{}
	for _, a := range addrs {
		parsed = append(parsed, netip.MustParseAddr(a))
	}
	restore := amai.SetEngineResolverForTest(func(_ context.Context, _ string) ([]netip.Addr, error) {
		return parsed, nil
	})
	t.Cleanup(restore)
}

func testCustomAI() *amai.AI {
	return &amai.AI{
		Identity:         commonidentity.Identity{ID: uuid.FromStringOrNil("dddddddd-0000-0000-0000-0000000000c1"), CustomerID: testEndpointCustomerID},
		EngineModel:      "customendpoint.my-model",
		EngineBaseURL:    testEndpointURL,
		EngineBaseKeyEnc: []byte("ciphertext-not-checked-here"),
		EngineBaseKey:    amai.RedactedString(testEndpointKey),
	}
}

func Test_resolveSessionEndpoint(t *testing.T) {
	deleted := time.Now()

	tests := []struct {
		name string

		llmType  pipecatcall.LLMType
		customer uuid.UUID
		mutate   func(a *amai.AI)
		nilAI    bool
		resolver []string

		expectNil bool
		expectErr bool
	}{
		{name: "custom model with a good live ai", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"}},
		{name: "keyless endpoint is fine", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"},
			mutate: func(a *amai.AI) { a.EngineBaseKeyEnc = nil; a.EngineBaseKey = "" }},
		{name: "other model type gets no endpoint even if the live ai has one", llmType: "openai.gpt-5", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"}, expectNil: true},
		{name: "nil live ai fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, nilAI: true, expectErr: true},
		{name: "live ai of another customer fails closed", llmType: "customendpoint.my-model", customer: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"), resolver: []string{"93.184.216.34"}, expectErr: true},
		{name: "soft deleted live ai fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"},
			mutate: func(a *amai.AI) { a.TMDelete = &deleted }, expectErr: true},
		{name: "empty url fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"},
			mutate: func(a *amai.AI) { a.EngineBaseURL = "" }, expectErr: true},
		{name: "ciphertext without plaintext fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"},
			mutate: func(a *amai.AI) { a.EngineBaseKey = "" }, expectErr: true},
		{name: "blocked answer fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"127.0.0.1"}, expectErr: true},
		{name: "one blocked answer among public ones fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34", "169.254.169.254"}, expectErr: true},
		{name: "http url fails closed", llmType: "customendpoint.my-model", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"},
			mutate: func(a *amai.AI) { a.EngineBaseURL = "http://llm.example-endpoint.test/v1" }, expectErr: true},
		{name: "malformed model id is rejected before any lookup", llmType: "customendpoint.", customer: testEndpointCustomerID, resolver: []string{"93.184.216.34"}, expectNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubEndpointResolver(t, tt.resolver...)

			var live *amai.AI
			if !tt.nilAI {
				live = testCustomAI()
				if tt.mutate != nil {
					tt.mutate(live)
				}
			}

			ep, err := resolveSessionEndpoint(context.Background(), tt.llmType, tt.customer, live)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got endpoint %v", ep)
				}
				if ep != nil {
					t.Errorf("an error must come with no endpoint")
				}
				if msg := err.Error(); strings.Contains(msg, "example-endpoint") || strings.Contains(msg, testEndpointKey) {
					t.Errorf("error text carries the url or key: %s", msg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.expectNil {
				if ep != nil {
					t.Fatalf("expected no endpoint, got %+v", ep)
				}
				return
			}
			if ep == nil || ep.BaseURL != live.EngineBaseURL || string(ep.BaseKey) != string(live.EngineBaseKey) {
				t.Fatalf("unexpected endpoint: %+v", ep)
			}
		})
	}
}

func Test_errEngineModelNotAvailable_customEndpointNotEchoed(t *testing.T) {
	for _, m := range []string{"customendpoint.secret-model-id", "  CustomEndpoint.x", "openrouter.x"} {
		if msg := errEngineModelNotAvailable(pipecatcall.LLMType(m)).Error(); msg != "engine model is not available" {
			t.Errorf("model %q echoed in %q", m, msg)
		}
	}
}

func Test_resolveSessionLLM_customEndpoint(t *testing.T) {
	typ, key, err := resolveSessionLLM("customendpoint.my-model", "legacy-engine-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != "customendpoint.my-model" || key != "" {
		t.Errorf("got (%q, %q): the legacy key must never be forwarded", typ, key)
	}
}

func Test_classifyPipelineError_connection(t *testing.T) {
	tests := []struct {
		raw    string
		expect message.ErrorCategory
	}{
		{"Error during completion: Connection error.", message.ErrorCategoryConnection},
		{"error during completion: connection error. extra", message.ErrorCategoryConnection},
		{"Error during completion: Request timed out.", message.ErrorCategoryTimeout},
		{"Error during completion: Error code: 401 - {'error': 'x'}", message.ErrorCategoryAuthentication},
		{"Error during completion: some other failure", message.ErrorCategoryUnknown},
	}
	for _, tt := range tests {
		if got := classifyPipelineError(tt.raw); got != tt.expect {
			t.Errorf("classifyPipelineError(%q) = %s, want %s", tt.raw, got, tt.expect)
		}
	}

	for _, hasSTT := range []bool{true, false} {
		if !shouldNotifyPipelineError(message.ErrorCategoryConnection, false, hasSTT) {
			t.Errorf("connection must be notified always (hasSTT=%v)", hasSTT)
		}
	}
}

func Test_SessionCreate_endpoint(t *testing.T) {
	h := &pipecatcallHandler{mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{}}
	pc := &pipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566bb"), CustomerID: testEndpointCustomerID}}

	se, err := h.SessionCreate(pc, pc.ID, nil, nil, "customendpoint.my-model", "", &sessionEndpoint{BaseURL: testEndpointURL, BaseKey: testEndpointKey})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer se.Cancel()

	if se.LLMBaseURL != testEndpointURL || string(se.LLMBaseKey) != testEndpointKey {
		t.Errorf("endpoint not stored on the session")
	}
	if !se.CustomEndpoint.Load() {
		t.Errorf("CustomEndpoint must be set for a customendpoint. session")
	}

	// The key reaches no formatted or marshaled rendering of the session.
	b, errMarshal := json.Marshal(se)
	if errMarshal != nil {
		t.Fatalf("marshal: %v", errMarshal)
	}
	for _, out := range []string{string(b), fmt.Sprintf("%v", se.LLMBaseKey), fmt.Sprintf("%+v", se.LLMBaseKey), fmt.Sprintf("%#v", se.LLMBaseKey), fmt.Sprintf("%s", se.LLMBaseKey)} {
		if strings.Contains(out, testEndpointKey) || strings.Contains(out, "example-endpoint") {
			t.Errorf("rendering carries the url or key: %s", out)
		}
	}

	pc2 := &pipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566bc")}}
	se2, err := h.SessionCreate(pc2, pc2.ID, nil, nil, "openai.gpt-5", "k", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer se2.Cancel()
	if se2.CustomEndpoint.Load() || se2.LLMBaseURL != "" {
		t.Errorf("a plain session must not be marked as custom endpoint")
	}
}

func Test_runnerHandlePipelineError_customEndpointWithholdsText(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()
	orig := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(orig)

	h := pipecatcallHandler{}
	se := newPipelineErrorTestSession(true) // voice: unknown is logged, not published
	se.CustomEndpoint.Store(true)

	raw := "Unknown error occurred: 500 - {'echo': 'Bearer " + testEndpointKey + "'} at " + testEndpointURL
	h.runnerHandlePipelineError(se, raw, false)
	h.runnerHandlePipelineError(se, raw, false) // repeat path

	found := 0
	for _, e := range hook.AllEntries() {
		if e.Data["func"] != "runnerHandlePipelineError" {
			continue
		}
		found++
		line, _ := e.String()
		if strings.Contains(line, testEndpointKey) || strings.Contains(line, "example-endpoint") || strings.Contains(line, "Bearer") {
			t.Errorf("log entry carries the raw text: %s", line)
		}
		if e.Data["category"] != message.ErrorCategoryUnknown {
			t.Errorf("category must still be logged, got %v", e.Data["category"])
		}
	}
	if found != 2 {
		t.Errorf("expected 2 entries (first and repeat), got %d", found)
	}
}

func Test_runnerStartScript_customEndpoint(t *testing.T) {
	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566cc"), CustomerID: testEndpointCustomerID},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		LLMType:       "customendpoint.my-model",
	}

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockPython := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	h := &pipecatcallHandler{pythonRunner: mockPython, toolHandler: mockTool}
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return(nil)

	se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "customendpoint.my-model", LLMBaseURL: testEndpointURL, LLMBaseKey: testEndpointKey}
	mockPython.EXPECT().Start(
		gomock.Any(), pc.ID, "customendpoint.my-model", "",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		testEndpointCustomerID, testEndpointURL, testEndpointKey,
	).Return(nil)

	if err := h.runnerStartScript(pc, se); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_runnerStartScript_customEndpointTeam(t *testing.T) {
	teamID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-0000000000b1")
	memberID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-0000000000b1")
	aicallID := uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-0000000000b1")
	live := testCustomAI()

	stubEndpointResolver(t, "93.184.216.34")

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	mockPython := NewMockPythonRunner(mc)
	h := &pipecatcallHandler{requestHandler: mockReq, toolHandler: mockTool, pythonRunner: mockPython}

	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566dd"), CustomerID: testEndpointCustomerID},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   aicallID,
	}
	// The snapshot type of the start member is not custom, a later member is.
	se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5"}

	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeTeam, AssistanceID: teamID}, nil)
	mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amteam.Team{
		Identity:      commonidentity.Identity{ID: teamID},
		StartMemberID: memberID,
		Members:       []amteam.Member{{ID: memberID, Name: "one", AIID: live.ID}},
	}, nil)
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), live.ID).Return(live, nil)
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()

	mockPython.EXPECT().Start(
		gomock.Any(), pc.ID, gomock.Any(), "",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		testEndpointCustomerID, "", "",
	).DoAndReturn(func(_ context.Context, _ uuid.UUID, _ string, _ string, _ any, _, _, _, _, _ string,
		_ any, team *resolvedTeamData, _ any, _ bool, _ uuid.UUID, _, _ string) error {
		if team == nil || len(team.Members) != 1 {
			t.Fatalf("expected one member, got %+v", team)
		}
		m := team.Members[0].AI
		if m.LLMType != "customendpoint.my-model" || m.LLMBaseURL != testEndpointURL || string(m.LLMBaseKey) != testEndpointKey || m.EngineKey != "" {
			t.Errorf("member endpoint not forwarded as expected: %+v", m)
		}
		return nil
	})

	if err := h.runnerStartScript(pc, se); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !se.CustomEndpoint.Load() {
		t.Errorf("a team with a custom member must mark the session")
	}
}

func Test_resolveTeamForPython_customEndpointMembers(t *testing.T) {
	teamID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-0000000000d1")
	m1 := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-0000000000d1")
	m2 := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-0000000000d2")
	other := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	good := testCustomAI()
	good.ID = uuid.FromStringOrNil("dddddddd-0000-0000-0000-0000000000d1")
	foreign := testCustomAI()
	foreign.ID = uuid.FromStringOrNil("dddddddd-0000-0000-0000-0000000000d2")
	foreign.CustomerID = other
	plain := &amai.AI{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("dddddddd-0000-0000-0000-0000000000d3"), CustomerID: testEndpointCustomerID}, EngineModel: "openai.gpt-5", EngineKey: "dummy-openai-key"}

	stubEndpointResolver(t, "93.184.216.34")

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	h := &pipecatcallHandler{requestHandler: mockReq, toolHandler: mockTool}

	mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amteam.Team{
		Identity:      commonidentity.Identity{ID: teamID},
		StartMemberID: m1,
		Members: []amteam.Member{
			{ID: m1, Name: "good", AIID: good.ID},
			{ID: m2, Name: "foreign", AIID: foreign.ID},
			{ID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-0000000000d3"), Name: "plain", AIID: plain.ID},
		},
	}, nil)
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), good.ID).Return(good, nil)
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), foreign.ID).Return(foreign, nil)
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), plain.ID).Return(plain, nil)
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()

	res, err := h.resolveTeamForPython(context.Background(), &amaicall.AIcall{AssistanceType: amaicall.AssistanceTypeTeam, AssistanceID: teamID}, testEndpointCustomerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, f, p := res.Members[0].AI, res.Members[1].AI, res.Members[2].AI
	if g.LLMType != "customendpoint.my-model" || g.LLMBaseURL != testEndpointURL || string(g.LLMBaseKey) != testEndpointKey || g.EngineKey != "" {
		t.Errorf("good member: %+v", g)
	}
	if f.LLMType != "" || f.LLMBaseURL != "" || f.LLMBaseKey != "" || f.EngineKey != "" {
		t.Errorf("a member of another customer must carry the rejection signal and no endpoint: %+v", f)
	}
	if p.LLMType != "openai.gpt-5" || p.EngineKey != "dummy-openai-key" || p.LLMBaseURL != "" {
		t.Errorf("plain member changed: %+v", p)
	}

	if out := fmt.Sprintf("%v %+v", g, g); strings.Contains(out, testEndpointKey) {
		t.Errorf("fmt rendering of a member carries the key")
	}
}

func Test_pythonRunnerStart_requestBody(t *testing.T) {
	var gotBody []byte
	status := http.StatusOK
	respBody := "ok"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	defer srv.Close()

	// The runner address is a constant, so route the call through a transport that targets the stub.
	origClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(r)
	})}
	defer func() { httpClient = origClient }()

	r := &pythonRunner{}
	id := uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566ee")

	if err := r.Start(context.Background(), id, "customendpoint.my-model", "", nil, "", "", "", "", "", nil, nil, nil, false,
		testEndpointCustomerID, testEndpointURL, testEndpointKey); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(gotBody, &m); err != nil {
		t.Fatalf("body is not json: %v", err)
	}
	if m["customer_id"] != testEndpointCustomerID.String() || m["llm_base_url"] != testEndpointURL || m["llm_base_key"] != testEndpointKey {
		t.Errorf("request body lacks the endpoint fields: %v", m)
	}

	// Team requests send the fields empty and omit them.
	if err := r.Start(context.Background(), id, "openai.gpt-5", "k", nil, "", "", "", "", "", nil, nil, nil, false, testEndpointCustomerID, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m = map[string]any{}
	_ = json.Unmarshal(gotBody, &m)
	if _, ok := m["llm_base_url"]; ok {
		t.Errorf("empty url must be omitted")
	}
	if m["customer_id"] != testEndpointCustomerID.String() {
		t.Errorf("customer_id must always be sent")
	}

	// The response body of a failed request that carried an endpoint is not put into the error.
	status = http.StatusUnprocessableEntity
	respBody = "echo " + testEndpointKey
	err := r.Start(context.Background(), id, "customendpoint.my-model", "", nil, "", "", "", "", "", nil, nil, nil, false,
		testEndpointCustomerID, testEndpointURL, testEndpointKey)
	if err == nil || strings.Contains(err.Error(), testEndpointKey) {
		t.Errorf("error must exist and not carry the body: %v", err)
	}
	// A failed team request with a member endpoint is handled the same way.
	team := &resolvedTeamData{Members: []resolvedMemberData{{AI: resolvedAIData{LLMBaseURL: testEndpointURL, LLMBaseKey: testEndpointKey}}}}
	err = r.Start(context.Background(), id, "openai.gpt-5", "", nil, "", "", "", "", "", nil, team, nil, false, testEndpointCustomerID, "", "")
	if err == nil || strings.Contains(err.Error(), testEndpointKey) {
		t.Errorf("team error must exist and not carry the body: %v", err)
	}
	// Without an endpoint the existing behavior stays.
	err = r.Start(context.Background(), id, "openai.gpt-5", "", nil, "", "", "", "", "", nil, nil, nil, false, testEndpointCustomerID, "", "")
	if err == nil || !strings.Contains(err.Error(), "body:") {
		t.Errorf("a request without an endpoint keeps the body in the error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func Test_startReferenceTypeAIcall_customEndpoint(t *testing.T) {
	pcID := uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566ff")
	aicallID := uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-0000000000e1")
	live := testCustomAI()

	tests := []struct {
		name string

		live      *amai.AI
		getErr    error
		expectErr bool
	}{
		{name: "live ai with endpoint starts the runner with it", live: live},
		{name: "ai lookup failure fails the start closed", getErr: fmt.Errorf("boom"), expectErr: true},
		{name: "blocked host fails the start closed", live: func() *amai.AI { a := testCustomAI(); a.EngineBaseURL = "https://127.0.0.1/v1"; return a }(), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubEndpointResolver(t, "93.184.216.34")

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)
			mockPython := NewMockPythonRunner(mc)
			h := &pipecatcallHandler{
				requestHandler:        mockReq,
				toolHandler:           mockTool,
				pythonRunner:          mockPython,
				mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{},
				muPipecatcallSession:  sync.Mutex{},
			}

			pc := &pipecatcall.Pipecatcall{
				Identity:      commonidentity.Identity{ID: pcID, CustomerID: testEndpointCustomerID},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   aicallID,
				LLMType:       "customendpoint.my-model",
			}
			mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{
				Identity:       commonidentity.Identity{ID: aicallID},
				ReferenceType:  amaicall.ReferenceTypeTask,
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   live.ID,
			}, nil).AnyTimes()
			if tt.getErr != nil {
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), live.ID).Return(nil, tt.getErr).AnyTimes()
			} else {
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), live.ID).Return(tt.live, nil).AnyTimes()
			}
			mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()
			mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return(nil, nil).AnyTimes()

			started := make(chan [2]string, 1)
			if !tt.expectErr {
				mockPython.EXPECT().Start(
					gomock.Any(), pcID, gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(),
				).DoAndReturn(func(_ context.Context, _ uuid.UUID, llmType string, llmKey string, _ any, _, _, _, _, _ string,
					_ any, _ any, _ any, _ bool, _ uuid.UUID, url string, key string) error {
					if llmType != "customendpoint.my-model" || llmKey != "" {
						t.Errorf("unexpected runner type or legacy key: %q %q", llmType, llmKey)
					}
					started <- [2]string{url, key}
					return nil
				}).Times(1)
			}

			err := h.startReferenceTypeAIcall(context.Background(), pc)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if strings.Contains(err.Error(), "example-endpoint") || strings.Contains(err.Error(), testEndpointKey) || strings.Contains(err.Error(), "127.0.0.1") {
					t.Errorf("error carries the url or key: %v", err)
				}
				if len(h.mapPipecatcallSession) != 0 {
					t.Errorf("no session may be created when the endpoint is not available")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			select {
			case got := <-started:
				if got[0] != testEndpointURL || got[1] != testEndpointKey {
					t.Errorf("runner did not receive the endpoint: %v", got)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("python runner was never started")
			}
			if se, errSe := h.SessionGet(pcID); errSe == nil {
				se.Cancel()
			}
		})
	}
}

func Test_runGetLLMKey_customEndpointFailsClosed(t *testing.T) {
	// The reference type call path has no live AI, so a custom endpoint session cannot start there.
	h := &pipecatcallHandler{}
	pc := &pipecatcall.Pipecatcall{ReferenceType: pipecatcall.ReferenceTypeCall, LLMType: "customendpoint.my-model"}

	typ, key, err := h.runGetLLMKey(context.Background(), pc)
	if err == nil {
		t.Fatalf("expected the call path to fail closed for a custom endpoint session, got (%q, %q)", typ, key)
	}
}
