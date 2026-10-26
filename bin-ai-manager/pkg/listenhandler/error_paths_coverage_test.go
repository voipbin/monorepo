package listenhandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/analysis"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aiaudithandler"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/aiprompthistoryhandler"
	"monorepo/bin-ai-manager/pkg/aipromptproposalhandler"
	"monorepo/bin-ai-manager/pkg/analysishandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpoauthhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
	"monorepo/bin-ai-manager/pkg/teamhandler"
	"monorepo/bin-common-handler/models/sock"
)

// errPathMocks holds one strict mock per business handler. A mock without an
// EXPECT() fails the test if it is called, which is how the cases below prove
// that a rejected request never reaches the business layer.
type errPathMocks struct {
	ai        *aihandler.MockAIHandler
	aicall    *aicallhandler.MockAIcallHandler
	aiaudit   *aiaudithandler.MockAIAuditHandler
	history   *aiprompthistoryhandler.MockAIPromptHistoryHandler
	proposal  *aipromptproposalhandler.MockAIPromptProposalHandler
	summary   *summaryhandler.MockSummaryHandler
	team      *teamhandler.MockTeamHandler
	message   *messagehandler.MockMessageHandler
	part      *participanthandler.MockParticipantHandler
	analysis  *analysishandler.MockAnalysisHandler
	mcpServer *mcpserverhandler.MockMcpServerHandler
	mcpOAuth  *mcpoauthhandler.MockMcpOAuthHandler
}

func newErrPathHandler(t *testing.T) (*listenHandler, *errPathMocks) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &errPathMocks{
		ai:        aihandler.NewMockAIHandler(mc),
		aicall:    aicallhandler.NewMockAIcallHandler(mc),
		aiaudit:   aiaudithandler.NewMockAIAuditHandler(mc),
		history:   aiprompthistoryhandler.NewMockAIPromptHistoryHandler(mc),
		proposal:  aipromptproposalhandler.NewMockAIPromptProposalHandler(mc),
		summary:   summaryhandler.NewMockSummaryHandler(mc),
		team:      teamhandler.NewMockTeamHandler(mc),
		message:   messagehandler.NewMockMessageHandler(mc),
		part:      participanthandler.NewMockParticipantHandler(mc),
		analysis:  analysishandler.NewMockAnalysisHandler(mc),
		mcpServer: mcpserverhandler.NewMockMcpServerHandler(mc),
		mcpOAuth:  mcpoauthhandler.NewMockMcpOAuthHandler(mc),
	}

	h := &listenHandler{
		aiHandler:               m.ai,
		aicallHandler:           m.aicall,
		aiauditHandler:          m.aiaudit,
		aiprompthistoryHandler:  m.history,
		aipromptproposalHandler: m.proposal,
		summaryHandler:          m.summary,
		teamHandler:             m.team,
		messageHandler:          m.message,
		participantHandler:      m.part,
		analysisHandler:         m.analysis,
		mcpServerHandler:        m.mcpServer,
		mcpOAuthHandler:         m.mcpOAuth,
	}

	return h, m
}

const (
	errPathNilID  = "00000000-0000-0000-0000-000000000000"
	errPathGoodID = "a0000000-0000-0000-0000-0000000000a1"
)

var errPathBoom = errors.New("boom")

type errPathCase struct {
	name   string
	method sock.RequestMethod
	uri    string
	data   string

	// setup registers the single business call that the request is expected
	// to reach; nil means the request must be rejected before that.
	setup func(m *errPathMocks)

	expectStatus int
}

// filterCustomerIDMatcher matches a filter map that carries exactly the given
// customer id under the "customer_id" field key, so a list handler that drops
// or replaces the parsed filters is caught rather than passed by a bare
// gomock.Any().
type filterCustomerIDMatcher struct {
	field    string
	expectID uuid.UUID
}

func (m filterCustomerIDMatcher) Matches(x any) bool {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Map || len(v.MapKeys()) != 1 {
		return false
	}
	k := v.MapKeys()[0]
	if k.String() != m.field {
		return false
	}
	got, ok := v.MapIndex(k).Interface().(uuid.UUID)
	return ok && got == m.expectID
}

func (m filterCustomerIDMatcher) String() string {
	return fmt.Sprintf("is a filter map of {%q: %s}", m.field, m.expectID)
}

const errPathFilterCustomerID = "a0000000-0000-0000-0000-0000000000a1"

func errPathListCases(resource string, setup func(m *errPathMocks, err error, filter gomock.Matcher)) []errPathCase {
	uri := "/v1/" + resource + "?page_size=10"
	matcher := filterCustomerIDMatcher{field: "customer_id", expectID: uuid.FromStringOrNil(errPathFilterCustomerID)}

	return []errPathCase{
		{
			name: resource + " list: unparsable uri", method: sock.RequestMethodGet,
			uri: uri + "\x7f", expectStatus: 400,
		},
		{
			name: resource + " list: body is not json", method: sock.RequestMethodGet,
			uri: uri, data: `{bad`, expectStatus: 400,
		},
		{
			name: resource + " list: filter value of the wrong type", method: sock.RequestMethodGet,
			uri: uri, data: `{"customer_id":"not-a-uuid"}`, expectStatus: 400,
		},
		{
			name: resource + " list: business error", method: sock.RequestMethodGet,
			uri: uri, data: `{"customer_id":"` + errPathFilterCustomerID + `"}`,
			setup: func(m *errPathMocks) { setup(m, errPathBoom, matcher) }, expectStatus: 500,
		},
		{
			name: resource + " list: not found maps to 404", method: sock.RequestMethodGet,
			uri: uri, data: `{"customer_id":"` + errPathFilterCustomerID + `"}`,
			setup: func(m *errPathMocks) { setup(m, dbhandler.ErrNotFound, matcher) }, expectStatus: 404,
		},
	}
}

func Test_processRequest_ErrorPaths(t *testing.T) {
	anyCtx := gomock.Any()

	tests := []errPathCase{}

	tests = append(tests, errPathListCases("aicalls", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.aicall.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("ais", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.ai.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("aiaudits", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.aiaudit.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("aipromptproposals", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.proposal.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("summaries", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.summary.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("teams", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.team.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("messages", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.message.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)
	tests = append(tests, errPathListCases("mcp_servers", func(m *errPathMocks, err error, filter gomock.Matcher) {
		m.mcpServer.EXPECT().List(anyCtx, uint64(10), "", filter).Return(nil, err)
	})...)

	id := errPathGoodID

	tests = append(tests, []errPathCase{
		// aicalls
		{
			name: "aicalls post: body is not json", method: sock.RequestMethodPost, uri: "/v1/aicalls", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "aicalls post: business error", method: sock.RequestMethodPost, uri: "/v1/aicalls", data: `{}`,
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().Start(anyCtx, uuid.Nil, gomock.Any(), uuid.Nil, uuid.Nil, gomock.Any(), uuid.Nil).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aicalls get: nil id", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "aicalls get: unparsable uri", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + id + "?skip_cache=\x7f",
			expectStatus: 400,
		},
		{
			name: "aicalls get: not found", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + id,
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "aicalls get: skip cache business error", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + id + "?skip_cache=true",
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().GetSkipCache(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aicalls delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/aicalls/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "aicalls delete: business error", method: sock.RequestMethodDelete, uri: "/v1/aicalls/" + id,
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aicalls terminate: nil id", method: sock.RequestMethodPost, uri: "/v1/aicalls/" + errPathNilID + "/terminate",
			expectStatus: 400,
		},
		{
			name: "aicalls terminate: business error", method: sock.RequestMethodPost, uri: "/v1/aicalls/" + id + "/terminate",
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().ProcessTerminate(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "aicalls listen: nil id", method: sock.RequestMethodPost, uri: "/v1/aicalls/" + errPathNilID + "/listen",
			expectStatus: 400,
		},
		{
			name: "aicalls tool_execute: body is not json", method: sock.RequestMethodPost, uri: "/v1/aicalls/" + id + "/tool_execute",
			data: `{bad`, expectStatus: 400,
		},
		{
			name: "aicalls tool_execute: business error", method: sock.RequestMethodPost, uri: "/v1/aicalls/" + id + "/tool_execute",
			data: `{"id":"tool-1","type":"function","function":{"name":"fn","arguments":"{}"},"pipecatcall_id":"b0000000-0000-0000-0000-0000000000b1"}`,
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().ToolHandle(anyCtx, uuid.FromStringOrNil(id), "tool-1", message.ToolType("function"), message.FunctionCall{Name: "fn", Arguments: "{}"}, uuid.FromStringOrNil("b0000000-0000-0000-0000-0000000000b1")).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aicalls tools mcp: business error", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + id + "/tools/mcp",
			setup: func(m *errPathMocks) {
				m.aicall.EXPECT().ResolveMcpTools(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},

		// ais
		{
			name: "ais post: body is not json", method: sock.RequestMethodPost, uri: "/v1/ais", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "ais post: mcp server whitelist rejected before create", method: sock.RequestMethodPost, uri: "/v1/ais",
			data: `{"mcp_server_ids":[]}`,
			setup: func(m *errPathMocks) {
				m.ai.EXPECT().ValidateMcpServerIDs(anyCtx, uuid.Nil, gomock.Any(), nil).Return(dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "ais get: nil id", method: sock.RequestMethodGet, uri: "/v1/ais/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "ais get: not found", method: sock.RequestMethodGet, uri: "/v1/ais/" + id,
			setup: func(m *errPathMocks) {
				m.ai.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "ais delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/ais/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "ais delete: business error", method: sock.RequestMethodDelete, uri: "/v1/ais/" + id,
			setup:        func(m *errPathMocks) { m.ai.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom) },
			expectStatus: 500,
		},
		{
			name: "ais put: body is not json", method: sock.RequestMethodPut, uri: "/v1/ais/" + id, data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "ais put: nil id", method: sock.RequestMethodPut, uri: "/v1/ais/" + errPathNilID, data: `{}`,
			expectStatus: 400,
		},
		{
			name: "ais put: current ai cannot be read for whitelist validation", method: sock.RequestMethodPut, uri: "/v1/ais/" + id,
			data: `{"mcp_server_ids":[]}`,
			setup: func(m *errPathMocks) {
				m.ai.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "ais activate insight: nil id", method: sock.RequestMethodPost, uri: "/v1/ais/" + errPathNilID + "/activate_insight",
			expectStatus: 400,
		},
		{
			name: "ais activate insight: business error", method: sock.RequestMethodPost, uri: "/v1/ais/" + id + "/activate_insight",
			setup: func(m *errPathMocks) {
				m.ai.EXPECT().ActivateInsight(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},

		// aiaudits
		{
			name: "aiaudits post: body is not json", method: sock.RequestMethodPost, uri: "/v1/aiaudits", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "aiaudits get: nil id", method: sock.RequestMethodGet, uri: "/v1/aiaudits/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "aiaudits get: not found", method: sock.RequestMethodGet, uri: "/v1/aiaudits/" + id,
			setup: func(m *errPathMocks) {
				m.aiaudit.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "aiaudits delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/aiaudits/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "aiaudits delete: business error", method: sock.RequestMethodDelete, uri: "/v1/aiaudits/" + id,
			setup: func(m *errPathMocks) {
				m.aiaudit.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},

		// aipromptproposals
		{
			name: "aipromptproposals post: body is not json", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "aipromptproposals get: nil id", method: sock.RequestMethodGet, uri: "/v1/aipromptproposals/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "aipromptproposals get: not found", method: sock.RequestMethodGet, uri: "/v1/aipromptproposals/" + id,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "aipromptproposals accept: nil id", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + errPathNilID + "/accept",
			data: `{}`, expectStatus: 400,
		},
		{
			name: "aipromptproposals accept: body is not json", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + id + "/accept",
			data: `{bad`, expectStatus: 400,
		},
		{
			name: "aipromptproposals reject: nil id", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + errPathNilID + "/reject",
			data: `{}`, expectStatus: 400,
		},
		{
			name: "aipromptproposals reject: body is not json", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + id + "/reject",
			data: `{bad`, expectStatus: 400,
		},
		{
			name: "aipromptproposals reject: proposal not found", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + id + "/reject",
			data: `{}`,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Reject(anyCtx, uuid.Nil, uuid.FromStringOrNil(id)).Return(nil, errors.New("proposal not found"))
			},
			expectStatus: 404,
		},
		{
			name: "aipromptproposals reject: proposal not completed", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + id + "/reject",
			data: `{}`,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Reject(anyCtx, uuid.Nil, uuid.FromStringOrNil(id)).Return(nil, errors.New("proposal not completed"))
			},
			expectStatus: 409,
		},
		{
			name: "aipromptproposals reject: other error", method: sock.RequestMethodPost, uri: "/v1/aipromptproposals/" + id + "/reject",
			data: `{}`,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Reject(anyCtx, uuid.Nil, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "aipromptproposals delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/aipromptproposals/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "aipromptproposals delete: business error", method: sock.RequestMethodDelete, uri: "/v1/aipromptproposals/" + id,
			setup: func(m *errPathMocks) {
				m.proposal.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},

		// summaries
		{
			name: "summaries post: body is not json", method: sock.RequestMethodPost, uri: "/v1/summaries", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "summaries get: nil id", method: sock.RequestMethodGet, uri: "/v1/summaries/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "summaries get: not found", method: sock.RequestMethodGet, uri: "/v1/summaries/" + id,
			setup: func(m *errPathMocks) {
				m.summary.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "summaries regenerate: nil id", method: sock.RequestMethodPost, uri: "/v1/summaries/" + errPathNilID + "/regenerate",
			data: `{}`, expectStatus: 400,
		},
		{
			name: "summaries regenerate: body is not json", method: sock.RequestMethodPost, uri: "/v1/summaries/" + id + "/regenerate",
			data: `{bad`, expectStatus: 400,
		},
		{
			name: "summaries delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/summaries/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "summaries delete: business error", method: sock.RequestMethodDelete, uri: "/v1/summaries/" + id,
			setup: func(m *errPathMocks) {
				m.summary.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},

		// teams
		{
			name: "teams post: body is not json", method: sock.RequestMethodPost, uri: "/v1/teams", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "teams get: nil id", method: sock.RequestMethodGet, uri: "/v1/teams/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "teams get: not found", method: sock.RequestMethodGet, uri: "/v1/teams/" + id,
			setup: func(m *errPathMocks) {
				m.team.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "teams put: body is not json", method: sock.RequestMethodPut, uri: "/v1/teams/" + id, data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "teams put: nil id", method: sock.RequestMethodPut, uri: "/v1/teams/" + errPathNilID, data: `{}`,
			expectStatus: 400,
		},
		{
			name: "teams delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/teams/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "teams delete: business error", method: sock.RequestMethodDelete, uri: "/v1/teams/" + id,
			setup: func(m *errPathMocks) {
				m.team.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "teams direct hash regenerate: business error", method: sock.RequestMethodPost, uri: "/v1/teams/" + id + "/direct-hash-regenerate",
			setup: func(m *errPathMocks) {
				m.team.EXPECT().DirectHashRegenerate(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},

		// messages
		{
			name: "messages post: body is not json", method: sock.RequestMethodPost, uri: "/v1/messages", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "messages get: nil id", method: sock.RequestMethodGet, uri: "/v1/messages/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "messages get: not found", method: sock.RequestMethodGet, uri: "/v1/messages/" + id,
			setup: func(m *errPathMocks) {
				m.message.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},

		// mcp servers
		{
			name: "mcp_servers post: body is not json", method: sock.RequestMethodPost, uri: "/v1/mcp_servers", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "mcp_servers get: nil id", method: sock.RequestMethodGet, uri: "/v1/mcp_servers/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "mcp_servers get: not found", method: sock.RequestMethodGet, uri: "/v1/mcp_servers/" + id,
			setup: func(m *errPathMocks) {
				m.mcpServer.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "mcp_servers put: body is not json", method: sock.RequestMethodPut, uri: "/v1/mcp_servers/" + id, data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "mcp_servers put: nil id", method: sock.RequestMethodPut, uri: "/v1/mcp_servers/" + errPathNilID, data: `{}`,
			expectStatus: 400,
		},
		{
			name: "mcp_servers delete: nil id", method: sock.RequestMethodDelete, uri: "/v1/mcp_servers/" + errPathNilID,
			expectStatus: 400,
		},
		{
			name: "mcp_servers delete: business error", method: sock.RequestMethodDelete, uri: "/v1/mcp_servers/" + id,
			setup: func(m *errPathMocks) {
				m.mcpServer.EXPECT().Delete(anyCtx, uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "mcp_servers oauth start: body is not json", method: sock.RequestMethodPost, uri: "/v1/mcp_servers/oauth/start", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "mcp_servers oauth callback: unparsable uri", method: sock.RequestMethodGet, uri: "/v1/mcp_servers/oauth/callback?state=s\x7f",
			expectStatus: 400,
		},
		{
			name: "mcp_servers oauth callback: business error", method: sock.RequestMethodGet, uri: "/v1/mcp_servers/oauth/callback?state=s1",
			setup:        func(m *errPathMocks) { m.mcpOAuth.EXPECT().CallbackExists(anyCtx, "s1").Return(false, errPathBoom) },
			expectStatus: 500,
		},
		{
			name: "mcp_servers oauth complete: body is not json", method: sock.RequestMethodPost, uri: "/v1/mcp_servers/oauth/complete", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "mcp_servers oauth complete: business error", method: sock.RequestMethodPost, uri: "/v1/mcp_servers/oauth/complete", data: `{}`,
			setup:        func(m *errPathMocks) { m.mcpOAuth.EXPECT().Complete(anyCtx, uuid.Nil, "", "").Return(nil, errPathBoom) },
			expectStatus: 500,
		},

		// services
		{
			name: "services aicall: body is not json", method: sock.RequestMethodPost, uri: "/v1/services/type/aicall", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "services summary: body is not json", method: sock.RequestMethodPost, uri: "/v1/services/type/summary", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "services task: body is not json", method: sock.RequestMethodPost, uri: "/v1/services/type/task", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "services analysis: body is not json", method: sock.RequestMethodPost, uri: "/v1/services/type/analysis", data: `{bad`,
			expectStatus: 400,
		},
		{
			name: "services analysis: business error", method: sock.RequestMethodPost, uri: "/v1/services/type/analysis", data: `{"prompt":"p","model":"m1"}`,
			setup: func(m *errPathMocks) {
				m.analysis.EXPECT().Run(anyCtx, &analysis.Request{Prompt: "p", Model: "m1"}).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},

		// participants and prompt histories
		{
			name: "aicall participants: nil id", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + errPathNilID + "/participants",
			expectStatus: 400,
		},
		{
			name: "aicall participants: business error", method: sock.RequestMethodGet, uri: "/v1/aicalls/" + id + "/participants?page_size=5",
			setup: func(m *errPathMocks) {
				m.part.EXPECT().ListByAIcallID(anyCtx, uuid.FromStringOrNil(id), uint64(5), "").Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "ai participants: nil id", method: sock.RequestMethodGet, uri: "/v1/ais/" + errPathNilID + "/participants",
			expectStatus: 400,
		},
		{
			name: "ai participants: business error", method: sock.RequestMethodGet, uri: "/v1/ais/" + id + "/participants?page_size=5",
			setup: func(m *errPathMocks) {
				m.part.EXPECT().ListByAIID(anyCtx, uuid.FromStringOrNil(id), uint64(5), "").Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
		{
			name: "prompt histories list: nil id", method: sock.RequestMethodGet, uri: "/v1/ais/" + errPathNilID + "/prompt_histories?page_size=5",
			expectStatus: 400,
		},
		{
			name: "prompt histories list: business error", method: sock.RequestMethodGet, uri: "/v1/ais/" + id + "/prompt_histories?page_size=5",
			setup: func(m *errPathMocks) {
				m.history.EXPECT().List(anyCtx, uuid.FromStringOrNil(id), uint64(5), "").Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "prompt histories get: nil id", method: sock.RequestMethodGet, uri: "/v1/ais/" + errPathNilID + "/prompt_histories/" + id,
			expectStatus: 400,
		},
		{
			name: "prompt histories get: business error", method: sock.RequestMethodGet, uri: "/v1/ais/" + id + "/prompt_histories/" + id,
			setup: func(m *errPathMocks) {
				m.history.EXPECT().Get(anyCtx, uuid.FromStringOrNil(id), uuid.FromStringOrNil(id)).Return(nil, errPathBoom)
			},
			expectStatus: 500,
		},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newErrPathHandler(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: []byte(tt.data)})
			if err != nil {
				t.Fatalf("processRequest must report failures through the response, got error: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("status mismatch. expected: %d, got: %+v", tt.expectStatus, res)
			}
		})
	}
}

// Test_processV1_ShortURI calls the item handlers directly with a path that
// has too few segments. The router never produces such a path, so this pins
// the guard each handler applies before it indexes into the path.
func Test_processV1_ShortURI(t *testing.T) {
	h, _ := newErrPathHandler(t)
	ctx := context.Background()

	handlers := map[string]func(context.Context, *sock.Request) (*sock.Response, error){
		"aicalls get":                  h.processV1AIcallsIDGet,
		"aicalls delete":               h.processV1AIcallsIDDelete,
		"aicalls terminate":            h.processV1AIcallsIDTerminatePost,
		"aicalls listen":               h.processV1AIcallsIDListenPost,
		"aicalls tool_execute":         h.processV1AIcallsIDToolExecutePost,
		"aicalls tools mcp":            h.processV1AIcallsIDToolsMcpGet,
		"aicalls participants":         h.processV1AIcallsIDParticipantsGet,
		"ais get":                      h.processV1AIsIDGet,
		"ais delete":                   h.processV1AIsIDDelete,
		"ais put":                      h.processV1AIsIDPut,
		"ais participants":             h.processV1AIsIDParticipantsGet,
		"ais activate insight":         h.processV1AIsIDActivateInsightPost,
		"ais direct hash regenerate":   h.processV1AIsIDDirectHashRegenerate,
		"ais prompt histories list":    h.processV1AIsIDPromptHistoriesGet,
		"ais prompt histories get":     h.processV1AIsIDPromptHistoriesIDGet,
		"aiaudits get":                 h.processV1AIAuditsIDGet,
		"aiaudits delete":              h.processV1AIAuditsIDDelete,
		"aipromptproposals get":        h.processV1AIPromptProposalsIDGet,
		"aipromptproposals accept":     h.processV1AIPromptProposalsIDAcceptPost,
		"aipromptproposals reject":     h.processV1AIPromptProposalsIDRejectPost,
		"aipromptproposals delete":     h.processV1AIPromptProposalsIDDelete,
		"summaries get":                h.processV1SummariesIDGet,
		"summaries regenerate":         h.processV1SummariesIDRegeneratePost,
		"summaries delete":             h.processV1SummariesIDDelete,
		"teams get":                    h.processV1TeamsIDGet,
		"teams put":                    h.processV1TeamsIDPut,
		"teams delete":                 h.processV1TeamsIDDelete,
		"teams direct hash regenerate": h.processV1TeamsIDDirectHashRegenerate,
		"messages get":                 h.processV1MessagesIDGet,
		"mcp_servers get":              h.processV1McpServersIDGet,
		"mcp_servers put":              h.processV1McpServersIDPut,
		"mcp_servers delete":           h.processV1McpServersIDDelete,
	}

	for name, fn := range handlers {
		t.Run(name, func(t *testing.T) {
			res, err := fn(ctx, &sock.Request{URI: "/v1/x", Method: sock.RequestMethodGet, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("expected 400 for a path without an id segment, got: %+v", res)
			}
		})
	}
}

func Test_errorResponse_WrappedNotFound(t *testing.T) {
	// a sentinel wrapped with fmt.Errorf %w keeps its 404 mapping.
	res := errorResponse(fmt.Errorf("wrapped: %w", dbhandler.ErrNotFound))
	if res.StatusCode != 404 {
		t.Errorf("expected 404 for a wrapped not-found error, got: %d", res.StatusCode)
	}
}
