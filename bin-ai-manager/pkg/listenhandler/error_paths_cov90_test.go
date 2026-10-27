package listenhandler

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/analysis"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"
)

const (
	cov90ID1 = "b0000000-0000-0000-0000-0000000000b1"
	cov90ID2 = "b0000000-0000-0000-0000-0000000000b2"
)

// Test_processRequest_WriteRouteFailures verifies the status that a request
// gets when the business layer fails or refuses it, for the write routes whose
// failure branch had no test.
func Test_processRequest_WriteRouteFailures(t *testing.T) {
	a := gomock.Any()
	idMcp := uuid.FromStringOrNil(cov90ID2)

	tests := []errPathCase{
		{
			name: "ais post: attaching the mcp servers fails after create", method: sock.RequestMethodPost, uri: "/v1/ais",
			data: `{"customer_id":"` + cov90ID1 + `","mcp_server_ids":["` + cov90ID2 + `"]}`,
			setup: func(m *errPathMocks) {
				created := &ai.AI{}
				created.ID = uuid.FromStringOrNil(cov90ID1)
				m.ai.EXPECT().ValidateMcpServerIDs(a, uuid.FromStringOrNil(cov90ID1), []uuid.UUID{idMcp}, nil).Return(nil)
				m.ai.EXPECT().Create(a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a).Return(created, nil)
				m.ai.EXPECT().UpdateMcpServerIDs(a, created.ID, []uuid.UUID{idMcp}).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "ais put: attaching the mcp servers fails after update", method: sock.RequestMethodPut, uri: "/v1/ais/" + cov90ID1,
			data: `{"mcp_server_ids":["` + cov90ID2 + `"]}`,
			setup: func(m *errPathMocks) {
				stored := &ai.AI{}
				stored.ID = uuid.FromStringOrNil(cov90ID1)
				updated := &ai.AI{}
				updated.ID = uuid.FromStringOrNil(cov90ID1)
				m.ai.EXPECT().Get(a, uuid.FromStringOrNil(cov90ID1)).Return(stored, nil)
				m.ai.EXPECT().ValidateMcpServerIDs(a, stored.CustomerID, []uuid.UUID{idMcp}, stored.McpServerIDs).Return(nil)
				m.ai.EXPECT().Update(a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a).Return(updated, nil)
				m.ai.EXPECT().UpdateMcpServerIDs(a, updated.ID, []uuid.UUID{idMcp}).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "mcp servers post: create fails", method: sock.RequestMethodPost, uri: "/v1/mcp_servers",
			data: `{"customer_id":"` + cov90ID1 + `","name":"n"}`,
			setup: func(m *errPathMocks) {
				m.mcpServer.EXPECT().Create(a, uuid.FromStringOrNil(cov90ID1), "n", a, a, a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "mcp servers put: update fails", method: sock.RequestMethodPut, uri: "/v1/mcp_servers/" + cov90ID1,
			data: `{"name":"n"}`,
			setup: func(m *errPathMocks) {
				m.mcpServer.EXPECT().Update(a, uuid.FromStringOrNil(cov90ID1), a, a, a, a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "summaries post: start fails", method: sock.RequestMethodPost, uri: "/v1/summaries",
			data: `{"customer_id":"` + cov90ID1 + `","language":"en-US"}`,
			setup: func(m *errPathMocks) {
				m.summary.EXPECT().Start(a, uuid.FromStringOrNil(cov90ID1), a, a, a, a, "en-US").Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "summaries regenerate: regenerate fails", method: sock.RequestMethodPost, uri: "/v1/summaries/" + cov90ID1 + "/regenerate",
			data: `{"language":"ko-KR"}`,
			setup: func(m *errPathMocks) {
				m.summary.EXPECT().Regenerate(a, uuid.FromStringOrNil(cov90ID1), "ko-KR").Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "teams post: create fails", method: sock.RequestMethodPost, uri: "/v1/teams",
			data: `{"customer_id":"` + cov90ID1 + `","name":"t"}`,
			setup: func(m *errPathMocks) {
				m.team.EXPECT().Create(a, uuid.FromStringOrNil(cov90ID1), "t", a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "teams put: update fails", method: sock.RequestMethodPut, uri: "/v1/teams/" + cov90ID1,
			data: `{"name":"t"}`,
			setup: func(m *errPathMocks) {
				m.team.EXPECT().Update(a, uuid.FromStringOrNil(cov90ID1), a, a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "services aicall: start fails", method: sock.RequestMethodPost, uri: "/v1/services/type/aicall",
			data: `{"assistance_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().ServiceStart(a, a, uuid.FromStringOrNil(cov90ID1), a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "services summary: start fails", method: sock.RequestMethodPost, uri: "/v1/services/type/summary",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.summary.EXPECT().ServiceStart(a, uuid.FromStringOrNil(cov90ID1), a, a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "services task: start fails", method: sock.RequestMethodPost, uri: "/v1/services/type/task",
			data: `{"assistance_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().ServiceStartTypeTask(a, a, uuid.FromStringOrNil(cov90ID1), a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "services analysis: run fails", method: sock.RequestMethodPost, uri: "/v1/services/type/analysis",
			data: `{}`,
			setup: func(m *errPathMocks) {
				m.analysis.EXPECT().Run(a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aiaudits post: rate limited maps to 429", method: sock.RequestMethodPost, uri: "/v1/aiaudits",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.aiaudit.EXPECT().Create(a, a, a, a).Return(nil, errors.New("rate limit exceeded"))
			},
			expectStatus: 429,
		},
		{
			name: "aiaudits post: already progressing maps to 409", method: sock.RequestMethodPost, uri: "/v1/aiaudits",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.aiaudit.EXPECT().Create(a, a, a, a).Return(nil, errors.New("audit already progressing"))
			},
			expectStatus: 409,
		},
		{
			name: "aiaudits post: other failure maps to 500", method: sock.RequestMethodPost, uri: "/v1/aiaudits",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.aiaudit.EXPECT().Create(a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aipromptproposals post: other failure maps to 500", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Create(a, a, a, a, a).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aipromptproposals accept: other failure maps to 500", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + cov90ID1 + "/accept",
			data: `{"customer_id":"` + cov90ID2 + `"}`,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Accept(a, uuid.FromStringOrNil(cov90ID2), uuid.FromStringOrNil(cov90ID1)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "ai prompt histories: unparsable uri", method: sock.RequestMethodGet, uri: "/v1/ais/" + cov90ID1 + "/prompt_histories?page_size=10\x7f",
			expectStatus: 400,
		},
		{
			name: "aicall participants: unparsable uri", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + cov90ID1 + "/participants?page_size=10\x7f",
			expectStatus: 400,
		},
		{
			name: "ai participants: unparsable uri", method: sock.RequestMethodGet, uri: "/v1/ais/" + cov90ID1 + "/participants?page_size=10\x7f",
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newErrPathHandler(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: json.RawMessage(tt.data)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

// Test_processRequest_SerializedResponses verifies the status and body of the
// successful responses whose serialization had no test.
func Test_processRequest_SerializedResponses(t *testing.T) {
	a := gomock.Any()

	aiRes := &ai.AI{}
	aiRes.ID = uuid.FromStringOrNil(cov90ID1)
	teamRes := &team.Team{}
	teamRes.ID = uuid.FromStringOrNil(cov90ID1)
	auditRes := []*aiaudit.AIAudit{{}}
	auditRes[0].ID = uuid.FromStringOrNil(cov90ID2)
	analysisRes := &analysis.Response{Result: json.RawMessage(`{"ok":true}`), Model: "test-model", FinishReason: "stop"}

	tests := []struct {
		name         string
		method       sock.RequestMethod
		uri          string
		data         string
		setup        func(m *errPathMocks)
		expectStatus int
		expectBody   any
	}{
		{
			name: "ais direct hash regenerate", method: sock.RequestMethodPost, uri: "/v1/ais/" + cov90ID1 + "/direct-hash-regenerate",
			setup: func(m *errPathMocks) {
				m.ai.EXPECT().DirectHashRegenerate(a, uuid.FromStringOrNil(cov90ID1)).Return(aiRes, nil)
			},
			expectStatus: 200, expectBody: aiRes,
		},
		{
			name: "teams direct hash regenerate", method: sock.RequestMethodPost, uri: "/v1/teams/" + cov90ID1 + "/direct-hash-regenerate",
			setup: func(m *errPathMocks) {
				m.team.EXPECT().DirectHashRegenerate(a, uuid.FromStringOrNil(cov90ID1)).Return(teamRes, nil)
			},
			expectStatus: 200, expectBody: teamRes,
		},
		{
			name: "aiaudits list", method: sock.RequestMethodGet, uri: "/v1/aiaudits?page_size=10",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.aiaudit.EXPECT().List(a, uint64(10), "", a).Return(auditRes, nil)
			},
			expectStatus: 200, expectBody: auditRes,
		},
		{
			name: "services analysis", method: sock.RequestMethodPost, uri: "/v1/services/type/analysis",
			data: `{}`,
			setup: func(m *errPathMocks) {
				m.analysis.EXPECT().Run(a, a).Return(analysisRes, nil)
			},
			expectStatus: 200, expectBody: analysisRes,
		},
		{
			name: "services summary", method: sock.RequestMethodPost, uri: "/v1/services/type/summary",
			data: `{"customer_id":"` + cov90ID1 + `"}`,
			setup: func(m *errPathMocks) {
				m.summary.EXPECT().ServiceStart(a, uuid.FromStringOrNil(cov90ID1), a, a, a, a, a).Return(&service.Service{ID: uuid.FromStringOrNil(cov90ID2)}, nil)
			},
			expectStatus: 200, expectBody: &service.Service{ID: uuid.FromStringOrNil(cov90ID2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newErrPathHandler(t)
			tt.setup(m)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: json.RawMessage(tt.data)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
			}

			expectData, errMarshal := json.Marshal(tt.expectBody)
			if errMarshal != nil {
				t.Fatalf("could not marshal the expected body: %v", errMarshal)
			}
			if string(res.Data) != string(expectData) {
				t.Errorf("unexpected body.\nexpected: %s\ngot: %s", expectData, res.Data)
			}
		})
	}
}
