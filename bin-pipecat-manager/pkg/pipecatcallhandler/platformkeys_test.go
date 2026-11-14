package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	amai "monorepo/bin-ai-manager/models/ai"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-pipecat-manager/internal/configkeys"
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"
)

// fakeResolver records the key names it is asked for and answers with "fake-value-<name>".
type fakeResolver struct {
	mu    sync.Mutex
	calls [][]string
	ids   []uuid.UUID
}

func (f *fakeResolver) Resolve(_ context.Context, id uuid.UUID, names []string) map[string]amai.RedactedString {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, append([]string(nil), names...))
	f.ids = append(f.ids, id)

	out := map[string]amai.RedactedString{}
	for _, n := range names {
		out[n] = amai.RedactedString("fake-value-" + n)
	}

	return out
}

func Test_platformKeyNames(t *testing.T) {
	tests := []struct {
		name string

		llm []string
		stt []string
		tts []string

		expect []string
	}{
		{"openai", []string{"openai.gpt-5"}, nil, nil, []string{"openai.api_key"}},
		{"openai upper case", []string{"OpenAI.gpt-5"}, nil, nil, []string{"openai.api_key"}},
		{"grok", []string{"grok.grok-4"}, nil, nil, []string{"xai.api_key"}},
		{"gemini with colon", []string{"gemini:gemini-flash"}, nil, nil, []string{"google.api_key"}},
		{"platform openrouter", []string{"platform_openrouter.anthropic/claude-haiku-4.5"}, nil, nil, []string{"openrouter.api_key"}},
		{"local", []string{"local.vllm-a.model-x"}, nil, nil, []string{"local_llm.endpoints"}},
		{"customer openrouter has no platform key", []string{"openrouter.anthropic/claude"}, nil, nil, []string{}},
		{"custom endpoint has no platform key", []string{"customendpoint.my-model"}, nil, nil, []string{}},
		{"unknown service", []string{"unknown.model"}, nil, nil, []string{}},
		{"no separator", []string{"openai"}, nil, nil, []string{}},
		{"empty llm", []string{""}, nil, nil, []string{}},
		{"deepgram", nil, []string{"deepgram"}, nil, []string{"deepgram.api_key"}},
		{"google stt has no key", nil, []string{"google"}, nil, []string{}},
		{"cartesia", nil, nil, []string{"cartesia"}, []string{"cartesia.api_key"}},
		{"elevenlabs upper case", nil, nil, []string{"ElevenLabs"}, []string{"elevenlabs.api_key"}},
		{"google tts has no key", nil, nil, []string{"google"}, []string{}},
		{
			"union is sorted and de-duplicated",
			[]string{"openai.gpt-5", "openai.gpt-4o", "gemini.x"}, []string{"deepgram", "deepgram"}, []string{"cartesia", "elevenlabs"},
			[]string{"cartesia.api_key", "deepgram.api_key", "elevenlabs.api_key", "google.api_key", "openai.api_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platformKeyNames(tt.llm, tt.stt, tt.tts)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

// Every configuration key is reachable from some vendor, and the names are the ones of configkeys.
func Test_platformKeyNames_coversEveryKey(t *testing.T) {
	got := platformKeyNames(
		[]string{"openai.m", "grok.m", "gemini.m", "platform_openrouter.m", "local.e.m"},
		[]string{"deepgram"},
		[]string{"cartesia", "elevenlabs"},
	)

	expect := []string{
		configkeys.KeyOpenAIAPIKey, configkeys.KeyDeepgramAPIKey, configkeys.KeyCartesiaAPIKey,
		configkeys.KeyXAIAPIKey, configkeys.KeyElevenLabsAPIKey, configkeys.KeyGoogleAPIKey,
		configkeys.KeyOpenRouterAPIKey, configkeys.KeyLocalLLMEndpoints,
	}
	if len(got) != len(expect) {
		t.Fatalf("Wrong match. expect: %d names, got: %v", len(expect), got)
	}
	have := map[string]bool{}
	for _, n := range got {
		have[n] = true
	}
	for _, n := range expect {
		if !have[n] {
			t.Errorf("Wrong match. %s is not reachable", n)
		}
	}
}

func newKeyTestPC(stt, tts string) *pipecatcall.Pipecatcall {
	return &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7e1f0a52-1111-2222-3333-4444555566bb")},
		STTType:  pipecatcall.STTType(stt),
		TTSType:  pipecatcall.TTSType(tts),
	}
}

func teamMember(llm, stt, tts string) resolvedMemberData {
	return resolvedMemberData{AI: resolvedAIData{LLMType: llm, STTType: stt, TTSType: tts}}
}

// A single AI session and a team session of the same configuration reach the resolver with the
// same key names and get the same map: both go through the one function.
func Test_sessionPlatformKeys_singleAndTeamSameMap(t *testing.T) {
	pc := newKeyTestPC("deepgram", "cartesia")

	single := &fakeResolver{}
	hs := &pipecatcallHandler{platformKeys: single}
	gotSingle := hs.sessionPlatformKeys(context.Background(), pc, &pipecatcall.Session{LLMRunnerType: "openai.gpt-5"}, nil)

	team := &fakeResolver{}
	ht := &pipecatcallHandler{platformKeys: team}
	gotTeam := ht.sessionPlatformKeys(context.Background(), pc, &pipecatcall.Session{LLMRunnerType: "ignored.top-level"}, &resolvedTeamData{
		Members: []resolvedMemberData{teamMember("openai.gpt-5", "deepgram", "cartesia")},
	})

	if !reflect.DeepEqual(single.calls, team.calls) || len(single.calls) != 1 {
		t.Errorf("Wrong match. single: %v, team: %v", single.calls, team.calls)
	}
	if !reflect.DeepEqual(gotSingle, gotTeam) || len(gotSingle) != 3 {
		t.Errorf("Wrong match. single: %v, team: %v", gotSingle, gotTeam)
	}
	if single.ids[0] != pc.ID {
		t.Errorf("Wrong match. the resolver must get the pipecatcall id, got: %v", single.ids[0])
	}
}

// A vendor that the session does not use is never asked for.
func Test_sessionPlatformKeys_unusedVendorExcluded(t *testing.T) {
	f := &fakeResolver{}
	h := &pipecatcallHandler{platformKeys: f}

	got := h.sessionPlatformKeys(context.Background(), newKeyTestPC("deepgram", "cartesia"), &pipecatcall.Session{LLMRunnerType: "openai.gpt-5"}, nil)

	expect := []string{"cartesia.api_key", "deepgram.api_key", "openai.api_key"}
	if len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0], expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, f.calls)
	}
	for _, unused := range []string{"xai.api_key", "google.api_key", "elevenlabs.api_key", "openrouter.api_key", "local_llm.endpoints"} {
		if _, ok := got[unused]; ok {
			t.Errorf("Wrong match. %s must not be in the map", unused)
		}
	}
}

func Test_sessionPlatformKeys_teamUnion(t *testing.T) {
	tests := []struct {
		name string

		stt, tts string
		members  []resolvedMemberData

		expect []string
	}{
		{
			"union of the members",
			"deepgram", "cartesia",
			[]resolvedMemberData{teamMember("openai.gpt-5", "deepgram", "cartesia"), teamMember("gemini.x", "google", "elevenlabs")},
			[]string{"cartesia.api_key", "deepgram.api_key", "elevenlabs.api_key", "google.api_key", "openai.api_key"},
		},
		{
			"text mode: the stt and tts of the members are not used",
			"", "",
			[]resolvedMemberData{teamMember("openai.gpt-5", "deepgram", "cartesia")},
			[]string{"openai.api_key"},
		},
		{
			"a rejected member (empty llm type) is skipped",
			"deepgram", "",
			[]resolvedMemberData{teamMember("", "deepgram", "cartesia"), teamMember("grok.x", "", "")},
			[]string{"deepgram.api_key", "xai.api_key"},
		},
		{
			"customer endpoint and customer openrouter members need no platform key",
			"", "",
			[]resolvedMemberData{teamMember("customendpoint.m", "", ""), teamMember("openrouter.x", "", "")},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeResolver{}
			h := &pipecatcallHandler{platformKeys: f}

			// the llm type of the session itself is not read for a team
			got := h.sessionPlatformKeys(context.Background(), newKeyTestPC(tt.stt, tt.tts), &pipecatcall.Session{LLMRunnerType: "local.never.read"},
				&resolvedTeamData{Members: tt.members})

			if tt.expect == nil {
				if len(f.calls) != 0 || got != nil {
					t.Errorf("Wrong match. expect: no resolver call and nil, got: %v %v", f.calls, got)
				}
				return
			}
			if len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0], tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, f.calls)
			}
		})
	}
}

// OpenRouter (platform) and local never take a customer key: the customer key of the session is
// not in the map, whose values come from the resolver only. A customer's own OpenRouter model gets
// no platform key at all.
func Test_sessionPlatformKeys_noCustomerKeyPenetration(t *testing.T) {
	const customer = "customer-key-sentinel"

	for _, llm := range []string{"platform_openrouter.model", "local.endpoint.model", "openrouter.model"} {
		t.Run(llm, func(t *testing.T) {
			f := &fakeResolver{}
			h := &pipecatcallHandler{platformKeys: f}

			got := h.sessionPlatformKeys(context.Background(), newKeyTestPC("", ""), &pipecatcall.Session{LLMRunnerType: llm, LLMKey: customer}, nil)

			body, err := json.Marshal(got)
			if err != nil || strings.Contains(string(body), customer) {
				t.Errorf("Wrong match. the customer key reached the map: %s %v", body, err)
			}
			if strings.HasPrefix(llm, "openrouter.") && len(got) != 0 {
				t.Errorf("Wrong match. a customer openrouter model must get no platform key, got: %v", got)
			}
		})
	}

	// a team member with a customer key of its own
	f := &fakeResolver{}
	h := &pipecatcallHandler{platformKeys: f}
	m := teamMember("platform_openrouter.model", "", "")
	m.AI.EngineKey = customer
	got := h.sessionPlatformKeys(context.Background(), newKeyTestPC("", ""), &pipecatcall.Session{}, &resolvedTeamData{Members: []resolvedMemberData{m}})
	if body, _ := json.Marshal(got); strings.Contains(string(body), customer) || len(got) != 1 {
		t.Errorf("Wrong match. got: %s", body)
	}
}

// A customer key does not change which keys the session asks for: the runner prefers the customer
// key itself, and the platform key of the vendor is still handed over.
func Test_sessionPlatformKeys_customerKeyDoesNotChangeNames(t *testing.T) {
	f := &fakeResolver{}
	h := &pipecatcallHandler{platformKeys: f}

	h.sessionPlatformKeys(context.Background(), newKeyTestPC("", ""), &pipecatcall.Session{LLMRunnerType: "openai.gpt-5", LLMKey: "customer-key"}, nil)

	if len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0], []string{"openai.api_key"}) {
		t.Errorf("Wrong match. got: %v", f.calls)
	}
}

func Test_sessionPlatformKeys_resolverNil(t *testing.T) {
	h := &pipecatcallHandler{}
	if got := h.sessionPlatformKeys(context.Background(), newKeyTestPC("deepgram", "cartesia"), &pipecatcall.Session{LLMRunnerType: "openai.gpt-5"}, nil); got != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", got)
	}
}

func Test_WithPlatformKeyResolver(t *testing.T) {
	f := &fakeResolver{}
	h := NewPipecatcallHandler(nil, nil, nil, nil, "host", WithPlatformKeyResolver(f)).(*pipecatcallHandler)
	if h.platformKeys != f {
		t.Errorf("Wrong match. the resolver was not set")
	}

	h = NewPipecatcallHandler(nil, nil, nil, nil, "host").(*pipecatcallHandler)
	if h.platformKeys != nil {
		t.Errorf("Wrong match. without the option no resolver is set")
	}
}

// runnerStartScript hands the resolver result to the runner as it is, for a single AI session.
func Test_runnerStartScript_passesPlatformKeys(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockPython := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	f := &fakeResolver{}
	h := &pipecatcallHandler{pythonRunner: mockPython, toolHandler: mockTool, platformKeys: f}

	pc := newKeyTestPC("deepgram", "cartesia")
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{})

	expect := map[string]amai.RedactedString{
		"openai.api_key":   "fake-value-openai.api_key",
		"deepgram.api_key": "fake-value-deepgram.api_key",
		"cartesia.api_key": "fake-value-cartesia.api_key",
	}
	mockPython.EXPECT().Start(
		gomock.Any(), pc.ID, "openai.gpt-5", "customer-key",
		gomock.Any(), "deepgram", gomock.Any(), "cartesia", gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		expect,
	).Return(nil)

	se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5", LLMKey: "customer-key"}
	if err := h.runnerStartScript(pc, se); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(f.calls) != 1 {
		t.Errorf("Wrong match. expect: one resolve per start, got: %d", len(f.calls))
	}
}

// Without a resolver the runner gets no platform keys (nil), which is the behavior before.
func Test_runnerStartScript_noResolverSendsNil(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockPython := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	h := &pipecatcallHandler{pythonRunner: mockPython, toolHandler: mockTool}

	pc := newKeyTestPC("deepgram", "cartesia")
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{})
	mockPython.EXPECT().Start(
		gomock.Any(), pc.ID, gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Nil(),
	).Return(nil)

	if err := h.runnerStartScript(pc, &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5"}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// capturedRequest returns the body that the runner client sent.
func captureRunnerBody(t *testing.T, respStatus int, respBody string) (func(map[string]amai.RedactedString, string, string) error, *[]byte) {
	t.Helper()

	var sent []byte
	useTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		sent, _ = io.ReadAll(r.Body)
		return &http.Response{StatusCode: respStatus, Body: io.NopCloser(strings.NewReader(respBody)), Header: http.Header{}, Request: r}, nil
	})

	start := func(keys map[string]amai.RedactedString, llmType, llmKey string) error {
		return (&pythonRunner{}).Start(context.Background(), uuid.FromStringOrNil("a1b2c3d4-5002-4000-8000-000000000001"),
			llmType, llmKey, nil, "", "", "", "", "", nil, nil, nil, false, uuid.Nil, "", "", keys)
	}

	return start, &sent
}

// The request carries the keys in the JSON body in plain text, as a field of its own that is
// separate from the customer key.
func Test_pythonRunner_Start_platformKeysInBody(t *testing.T) {
	start, sent := captureRunnerBody(t, http.StatusOK, "")

	keys := map[string]amai.RedactedString{"openai.api_key": "platform-fake-1", "deepgram.api_key": "platform-fake-2"}
	if err := start(keys, "openai.gpt-5", "customer-fake"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var body struct {
		LLMKey       string            `json:"llm_key"`
		PlatformKeys map[string]string `json:"platform_keys"`
	}
	if err := json.Unmarshal(*sent, &body); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if body.LLMKey != "customer-fake" || len(body.PlatformKeys) != 2 ||
		body.PlatformKeys["openai.api_key"] != "platform-fake-1" || body.PlatformKeys["deepgram.api_key"] != "platform-fake-2" {
		t.Errorf("Wrong match. got: %+v", body)
	}
	if strings.Contains(string(*sent), "customer-fake") && strings.Count(string(*sent), "customer-fake") != 1 {
		t.Errorf("Wrong match. the customer key must be sent once, in its own field")
	}
	if body.PlatformKeys["openai.api_key"] == body.LLMKey {
		t.Errorf("Wrong match. the two keys must stay separate")
	}
}

func Test_pythonRunner_Start_platformKeysOmittedWhenEmpty(t *testing.T) {
	for name, keys := range map[string]map[string]amai.RedactedString{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			start, sent := captureRunnerBody(t, http.StatusOK, "")
			if err := start(keys, "openai.gpt-5", ""); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if strings.Contains(string(*sent), "platform_keys") {
				t.Errorf("Wrong match. the field must be omitted: %s", *sent)
			}
		})
	}
}

// The value type prints as redacted in every text form. Changing the map to plain strings would
// fail here.
func Test_platformKeys_valuesAreRedactedWhenPrinted(t *testing.T) {
	keys := map[string]amai.RedactedString{"openai.api_key": "platform-fake-print"}

	printed := fmt.Sprintf("%v|%+v|%#v|%s", keys, keys, keys, keys)
	if strings.Contains(printed, "platform-fake-print") || !strings.Contains(printed, "<redacted>") {
		t.Errorf("Wrong match. got: %s", printed)
	}
}

// A failed start does not put a key in the error text, whatever the runner answers.
func Test_pythonRunner_Start_errorsHoldNoKey(t *testing.T) {
	keys := map[string]amai.RedactedString{"openai.api_key": "platform-fake-err"}

	t.Run("non 200", func(t *testing.T) {
		start, _ := captureRunnerBody(t, http.StatusBadRequest, "echo platform-fake-err")
		err := start(keys, "openai.gpt-5", "")
		if err == nil || strings.Contains(err.Error(), "platform-fake-err") {
			t.Errorf("Wrong match. got: %v", err)
		}
	})

	t.Run("send failure", func(t *testing.T) {
		useTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("dial refused")
		})
		err := (&pythonRunner{}).Start(context.Background(), uuid.Nil, "openai.gpt-5", "", nil, "", "", "", "", "", nil, nil, nil, false, uuid.Nil, "", "", keys)
		if err == nil || strings.Contains(err.Error(), "platform-fake-err") {
			t.Errorf("Wrong match. got: %v", err)
		}
	})
}

// The whole runner start, with a failing runner, writes the key to no log line.
func Test_runnerStartScript_logsHoldNoKey(t *testing.T) {
	logs := logtest.New(t)

	useTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("echo platform-fake-log-openai.api_key")), Header: http.Header{}, Request: r}, nil
	})

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockTool := toolhandler.NewMockToolHandler(mc)
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()
	h := &pipecatcallHandler{pythonRunner: NewPythonRunner(), toolHandler: mockTool, platformKeys: &fakeResolver{}}

	pc := newKeyTestPC("deepgram", "cartesia")
	se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5"}

	logrus.SetLevel(logrus.DebugLevel)
	if err := h.runnerStartScript(pc, se); err == nil {
		t.Fatalf("Wrong match. expect: an error for the failing runner")
	}
	h.RunnerStart(pc, se)

	logs.AssertAbsent(t, "fake-value-", "platform-fake-log")
}
