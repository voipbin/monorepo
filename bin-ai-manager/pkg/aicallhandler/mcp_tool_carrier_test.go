package aicallhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

const carrierTestKeys = "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func reservedServer(id uuid.UUID) *mcpserver.McpServer {
	return &mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive, URL: "https://carrier.invalid/fake"}
}

func ordinaryServer(id uuid.UUID) *mcpserver.McpServer {
	return &mcpserver.McpServer{Identity: commonidentityFor(id), Status: mcpserver.StatusActive, URL: "https://mcp.example.com/"}
}

func testAI(ids ...uuid.UUID) *ai.AI {
	return &ai.AI{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: testCustomerID}, McpServerIDs: ids}
}

func Test_discoverMcpTools_reservedServersSkipTheSlots(t *testing.T) {
	defer shortenSlotWait(t, 100*time.Millisecond)()

	// every slot is held, as another customer's slow listings would
	for i := 0; i < cap(mcpDiscoverySlots); i++ {
		mcpDiscoverySlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(mcpDiscoverySlots); i++ {
			<-mcpDiscoverySlots
		}
	}()

	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	ordinary, reserved := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	srv.EXPECT().Get(gomock.Any(), ordinary).Return(ordinaryServer(ordinary), nil)
	srv.EXPECT().Get(gomock.Any(), reserved).Return(reservedServer(reserved), nil)
	tl.EXPECT().ListTools(gomock.Any(), reserved).Return([]mcptoolhandler.McpTool{{Name: "track_fake"}}, nil)
	// no ListTools expectation for the ordinary server: it has no slot

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	got := h.discoverMcpTools(context.Background(), testAI(ordinary, reserved), false)

	if len(got) != 1 || got[0].ref.ServerID != reserved || got[0].ref.ToolName != "track_fake" {
		t.Fatalf("the reserved server's tool must be found without a slot, got %+v", got)
	}
}

func Test_discoverMcpTools_reservedServersAreNotBlockedByAnEarlierSlowServer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	slow, reserved := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	srv.EXPECT().Get(gomock.Any(), slow).Return(ordinaryServer(slow), nil)
	srv.EXPECT().Get(gomock.Any(), reserved).Return(reservedServer(reserved), nil)
	tl.EXPECT().ListTools(gomock.Any(), slow).DoAndReturn(func(ctx context.Context, _ uuid.UUID) ([]mcptoolhandler.McpTool, error) {
		<-ctx.Done() // a server that uses the whole session start budget
		return nil, ctx.Err()
	})
	tl.EXPECT().ListTools(gomock.Any(), reserved).DoAndReturn(func(ctx context.Context, _ uuid.UUID) ([]mcptoolhandler.McpTool, error) {
		if ctx.Err() != nil {
			t.Error("the reserved server must be listed before the budget is spent")
		}
		return []mcptoolhandler.McpTool{{Name: "track_fake"}}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	got := h.discoverMcpTools(ctx, testAI(slow, reserved), false)

	if len(got) != 1 || got[0].ref.ServerID != reserved {
		t.Fatalf("got %+v", got)
	}
}

func Test_discoverMcpTools_reservedServersGoFirstAndOthersKeepTheirOrder(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	o1, r1, o2, r2 := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	for _, id := range []uuid.UUID{o1, o2} {
		srv.EXPECT().Get(gomock.Any(), id).Return(ordinaryServer(id), nil)
		tl.EXPECT().ListTools(gomock.Any(), id).Return([]mcptoolhandler.McpTool{{Name: "other"}}, nil)
	}
	for _, id := range []uuid.UUID{r1, r2} {
		srv.EXPECT().Get(gomock.Any(), id).Return(reservedServer(id), nil)
		tl.EXPECT().ListTools(gomock.Any(), id).Return([]mcptoolhandler.McpTool{{Name: "track_fake"}}, nil)
	}

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	got := h.discoverMcpTools(context.Background(), testAI(o1, r1, o2, r2), false)

	order := []uuid.UUID{}
	for _, d := range got {
		order = append(order, d.ref.ServerID)
	}
	want := []uuid.UUID{r1, r2, o1, o2}
	if len(order) != len(want) {
		t.Fatalf("got %d tools, want %d", len(order), len(want))
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func Test_discoverMcpTools_failuresAreSkippedAndLoggedWithoutDetail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	missing, failing, good := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	srv.EXPECT().Get(gomock.Any(), missing).Return(nil, errors.New("not found"))
	srv.EXPECT().Get(gomock.Any(), failing).Return(reservedServer(failing), nil)
	srv.EXPECT().Get(gomock.Any(), good).Return(reservedServer(good), nil)
	tl.EXPECT().ListTools(gomock.Any(), failing).Return(nil, errors.New("mcptoolhandler.ListTools: carrier integration is not available"))
	tl.EXPECT().ListTools(gomock.Any(), good).Return([]mcptoolhandler.McpTool{{Name: "track_fake"}}, nil)

	logs := logtest.New(t)
	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	got := h.discoverMcpTools(context.Background(), testAI(missing, failing, good), false)

	if len(got) != 1 || got[0].ref.ServerID != good {
		t.Fatalf("got %+v", got)
	}
	dump := logs.Dump()
	for _, want := range []string{"Could not get mcp server, skipping its tools", "Could not list tools from mcp server, skipping"} {
		if !strings.Contains(dump, want) {
			t.Errorf("the log must keep %q", want)
		}
	}
	logs.AssertAbsent(t, "carrier.invalid")
}

func Test_discoverMcpTools_inactiveReservedServerIsSkipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	id := uuid.Must(uuid.NewV4())
	s := reservedServer(id)
	s.Status = mcpserver.StatusDisabled
	srv.EXPECT().Get(gomock.Any(), id).Return(s, nil)
	// no ListTools expectation

	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	if got := h.discoverMcpTools(context.Background(), testAI(id), false); len(got) != 0 {
		t.Errorf("a disabled server must contribute nothing, got %+v", got)
	}
}

func Test_discoverMcpTools_toolLimitCountsUnlistedServers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	tl := mcptoolhandler.NewMockMcpToolHandler(mc)

	first, second := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	srv.EXPECT().Get(gomock.Any(), first).Return(reservedServer(first), nil)
	srv.EXPECT().Get(gomock.Any(), second).Return(ordinaryServer(second), nil)

	tools := make([]mcptoolhandler.McpTool, 0, mcpMaxToolsPerResolution)
	for i := 0; i < mcpMaxToolsPerResolution; i++ {
		tools = append(tools, mcptoolhandler.McpTool{Name: fmt.Sprintf("t%d", i)})
	}
	tl.EXPECT().ListTools(gomock.Any(), first).Return(tools, nil)
	// the second server is past the limit and is not listed

	logs := logtest.New(t)
	h := &aicallHandler{mcpServerHandler: srv, mcptoolHandler: tl}
	got := h.discoverMcpTools(context.Background(), testAI(first, second), false)

	if len(got) != mcpMaxToolsPerResolution {
		t.Fatalf("got %d tools", len(got))
	}
	if !strings.Contains(logs.Dump(), "1 whitelisted servers were not listed") {
		t.Errorf("the limit warning must count the servers not listed: %s", logs.Dump())
	}
}

// fakeCarrierHandler stands in for the adapter package.
type fakeCarrierHandler struct {
	text    string
	isError bool
	read    bool
}

func (f *fakeCarrierHandler) Tools(carrier string) ([]mcpserver.CarrierToolDef, bool) {
	if carrier != "fake" {
		return nil, false
	}
	return []mcpserver.CarrierToolDef{{
		Name:        "track_fake",
		Description: "Look up the status of a fake package.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"tracking_number":{"type":"string","description":"The number."}},"required":["tracking_number"]}`),
	}}, true
}

func (f *fakeCarrierHandler) Call(ctx context.Context, call mcpserver.CarrierCall) (string, bool, error) {
	if call.Secret != nil && f.read {
		if _, _, err := call.Secret(); err != nil {
			return "", false, err
		}
	}
	return f.text, f.isError, nil
}

// carrierStack wires the real mcptoolhandler between mocked storage and a
// fake adapter, so the discovery and dispatch paths of aicallhandler run
// against the real reserved-host branch.
func carrierStack(t *testing.T, mc *gomock.Controller, a *ai.AI, server *mcpserver.McpServer, f *fakeCarrierHandler) *aicallHandler {
	t.Helper()

	db := dbhandler.NewMockDBHandler(mc)
	db.EXPECT().McpServerGet(gomock.Any(), server.ID).Return(server, nil).AnyTimes()
	srv := mcpserverhandler.NewMockMcpServerHandler(mc)
	srv.EXPECT().Get(gomock.Any(), server.ID).Return(server, nil).AnyTimes()

	table := mcpserver.CarrierTable{"/fake": {Carrier: "fake", Credential: mcpserver.CredentialClientIDSecret}}
	tool, err := mcptoolhandler.NewMcpToolHandler(db, carrierTestKeys, 5, nil, table, f)
	if err != nil {
		t.Fatal(err)
	}

	aiH := aihandler.NewMockAIHandler(mc)
	aiH.EXPECT().Get(gomock.Any(), a.ID).Return(a, nil).AnyTimes()
	return &aicallHandler{aiHandler: aiH, mcpServerHandler: srv, mcptoolHandler: tool}
}

func Test_carrierTools_areDiscoveredAndDispatchedThroughBothPaths(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	server := reservedServer(uuid.Must(uuid.NewV4()))
	f := &fakeCarrierHandler{text: `{"status":"in_transit"}`}
	a := testAI(server.ID)
	h := carrierStack(t, mc, a, server, f)

	wantName := "mcp_" + mcpServerIDShort(server.ID) + "_track_fake"

	// the pipecat path: names, descriptions and normalized schemas
	tools, toolMap, err := h.resolveMcpOnly(context.Background(), a)
	if err != nil || len(tools) != 1 || len(toolMap) != 1 {
		t.Fatalf("resolveMcpOnly: %v %+v %+v", err, tools, toolMap)
	}
	if _, ok := toolMap[wantName]; !ok {
		t.Fatalf("tool map %v does not hold %s", toolMap, wantName)
	}
	if string(tools[0].Name) != wantName {
		t.Errorf("advertised name = %q", tools[0].Name)
	}

	// the session start path: names only
	startMap := h.resolveMcpToolMap(context.Background(), a)
	if len(startMap) != 1 || startMap[wantName].ToolName != "track_fake" {
		t.Fatalf("resolveMcpToolMap: %+v", startMap)
	}

	// dispatch: the same ToolHandle path as any MCP tool
	c := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   a.ID,
		Metadata:       map[string]any{aicall.MetaKeyMcpToolMap: startMap},
	}
	got := h.toolHandleMcpCall(context.Background(), c, &message.ToolCall{ID: "t1", Function: message.FunctionCall{Name: message.FunctionCallName(wantName), Arguments: `{"tracking_number":"X"}`}})
	if got.Result != "success" || !strings.Contains(got.Message, "in_transit") {
		t.Errorf("success dispatch: %+v", got)
	}

	// an error result keeps the fixed text the adapter gave
	f.text, f.isError = "No FAKE API key is saved in this MCP server's settings. Do not retry.", true
	got = h.toolHandleMcpCall(context.Background(), c, &message.ToolCall{ID: "t2", Function: message.FunctionCall{Name: message.FunctionCallName(wantName), Arguments: `{}`}})
	if got.Result != "failed" || got.Message != f.text {
		t.Errorf("error dispatch: %+v", got)
	}
}

func Test_carrierTools_aKeyNeverReachesTheLogsOrTheMessage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	const secret = "SENTINEL-ID:SENTINEL-SECRET"
	crypto, err := mcpserverhandler.NewSecretCrypto(carrierTestKeys)
	if err != nil {
		t.Fatal(err)
	}
	ct, nonce, version, err := crypto.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}

	for _, corrupt := range []bool{false, true} {
		name := "decryptable"
		if corrupt {
			name = "undecryptable"
		}
		t.Run(name, func(t *testing.T) {
			server := reservedServer(uuid.Must(uuid.NewV4()))
			server.AuthType = mcpserver.AuthTypeBearer
			server.SecretCiphertext = append([]byte(nil), ct...)
			server.SecretNonce, server.KeyVersion = nonce, version
			if corrupt {
				server.SecretCiphertext[0] ^= 0xff
			}

			f := &fakeCarrierHandler{text: "{}", read: true}
			a := testAI(server.ID)
			h := carrierStack(t, mc, a, server, f)
			wantName := "mcp_" + mcpServerIDShort(server.ID) + "_track_fake"
			toolMap := h.resolveMcpToolMap(context.Background(), a)

			c := &aicall.AIcall{
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   a.ID,
				Metadata:       map[string]any{aicall.MetaKeyMcpToolMap: toolMap},
			}
			logs := logtest.New(t)
			got := h.toolHandleMcpCall(context.Background(), c, &message.ToolCall{ID: "t", Function: message.FunctionCall{Name: message.FunctionCallName(wantName), Arguments: `{}`}})

			if corrupt && (got.Result != "failed" || got.Message != "MCP tool call failed") {
				t.Errorf("a decrypt fault must reach the model as the fixed failure text, got %+v", got)
			}
			for _, out := range []string{got.Message, got.Result} {
				if strings.Contains(out, "SENTINEL") {
					t.Errorf("a key leaked into %q", out)
				}
			}
			logs.AssertAbsent(t, "SENTINEL", string(server.SecretCiphertext))
		})
	}
}
