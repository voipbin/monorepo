package server

import (
	"net/http"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaiprompthistory "monorepo/bin-ai-manager/models/aiprompthistory"
	amparticipant "monorepo/bin-ai-manager/models/participant"
	amteam "monorepo/bin-ai-manager/models/team"
	amtool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	wcsession "monorepo/bin-webchat-manager/models/session"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Teams_ListAndCrud(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetTeams", path: "/teams",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().TeamGetsByCustomerID(gomock.Any(), a, size, token)
		},
		mk: s4MkList[amteam.WebhookMessage],
	})...)

	ret := s4Obj[amteam.WebhookMessage](s4ID1, nil)
	bad := "/teams/not-a-uuid"
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetTeamsId", method: http.MethodGet, path: "/teams/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().TeamGet(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteTeamsId", method: http.MethodDelete, path: "/teams/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().TeamDelete(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostTeamsIdDirectHashRegenerate", method: http.MethodPost, path: "/teams/" + s4ID1 + "/direct-hash-regenerate",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().TeamDirectHashRegenerate(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)

	createBody := `{"name":"team-a","detail":"team detail","start_member_id":"` + s4ID1 + `","parameter":{"retries":3},` +
		`"members":[{"id":"` + s4ID2 + `","name":"member-a","ai_id":"` + s4ID3 + `","transitions":[{"function_name":"handoff","description":"go to b","next_member_id":"` + s4ID1 + `"}]},` +
		`{"id":"` + s4ID1 + `","name":"member-b","ai_id":"` + s4ID3 + `"}]}`
	wantMembers := []amteam.Member{
		{
			ID: s4UUID(s4ID2), Name: "member-a", AIID: s4UUID(s4ID3),
			Transitions: []amteam.Transition{{FunctionName: "handoff", Description: "go to b", NextMemberID: s4UUID(s4ID1)}},
		},
		{ID: s4UUID(s4ID1), Name: "member-b", AIID: s4UUID(s4ID3)},
	}
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostTeams", method: http.MethodPost, path: "/teams", body: createBody, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().TeamCreate(gomock.Any(), a, "team-a", "team detail", s4UUID(s4ID1), wantMembers, map[string]any{"retries": float64(3)})
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PostTeams/no_parameter_no_members", method: http.MethodPost, path: "/teams",
		body: `{"name":"n","detail":"d","start_member_id":"not-a-uuid","members":[]}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().TeamCreate(gomock.Any(), a, "n", "d", uuid.Nil, []amteam.Member{}, map[string]any(nil)).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})

	// PutTeamsId
	name, detail := "new-name", "new-detail"
	startID := s4UUID(s4ID1)
	nilID := uuid.Nil
	params := map[string]any{"k": "v"}
	members := wantMembers
	putBody := `{"name":"new-name","detail":"new-detail","start_member_id":"` + s4ID1 + `","parameter":{"k":"v"},` +
		`"members":[{"id":"` + s4ID2 + `","name":"member-a","ai_id":"` + s4ID3 + `","transitions":[{"function_name":"handoff","description":"go to b","next_member_id":"` + s4ID1 + `"}]},` +
		`{"id":"` + s4ID1 + `","name":"member-b","ai_id":"` + s4ID3 + `"}]}`
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutTeamsId", method: http.MethodPut, path: "/teams/" + s4ID2, body: putBody, badJSON: true,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().TeamUpdate(gomock.Any(), a, s4UUID(s4ID2), &name, &detail, &startID, &members, &params)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases,
		s4Case{
			name: "PutTeamsId/empty_start_member_id_clears_to_nil_uuid", method: http.MethodPut, path: "/teams/" + s4ID2,
			body: `{"start_member_id":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TeamUpdate(gomock.Any(), a, s4UUID(s4ID2), (*string)(nil), (*string)(nil), &nilID, (*[]amteam.Member)(nil), (*map[string]any)(nil)).Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
		s4Case{
			name: "PutTeamsId/invalid_start_member_id", method: http.MethodPut, path: "/teams/" + s4ID2,
			body:          `{"start_member_id":"not-a-uuid"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_START_MEMBER_ID",
		},
		s4Case{
			name: "PutTeamsId/empty_body_object_passes_all_nil", method: http.MethodPut, path: "/teams/" + s4ID2,
			body: `{}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TeamUpdate(gomock.Any(), a, s4UUID(s4ID2), (*string)(nil), (*string)(nil), (*uuid.UUID)(nil), (*[]amteam.Member)(nil), (*map[string]any)(nil)).Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
	)
	s4RunCases(t, cases)
}

func Test_WebchatSessions_Matrix(t *testing.T) {
	var cases []s4Case
	widget := s4UUID(s4ID3)
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetWebchatSessions", path: "/webchat_sessions",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().WebchatSessionList(gomock.Any(), a, size, token, uuid.Nil)
		},
		mk: s4MkList[wcsession.WebhookMessage],
	})...)
	cases = append(cases, s4Case{
		name: "GetWebchatSessions/widget_filter_passthrough", method: http.MethodGet, path: "/webchat_sessions?widget_id=" + s4ID3,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatSessionList(gomock.Any(), a, uint64(100), "", widget).Return(s4MkList[wcsession.WebhookMessage](), nil)
		},
		wantStatus: 200, wantBody: map[string]any{"result": []any{}, "next_page_token": ""},
	})

	ret := s4Obj[wcsession.WebhookMessage](s4ID1, nil)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetWebchatSessionsId", method: http.MethodGet, path: "/webchat_sessions/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().WebchatSessionGet(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteWebchatSessionsId", method: http.MethodDelete, path: "/webchat_sessions/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().WebchatSessionDelete(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostWebchatSessionsIdEnd", method: http.MethodPost, path: "/webchat_sessions/" + s4ID1 + "/end",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().WebchatSessionEnd(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostWebchatSessions", method: http.MethodPost, path: "/webchat_sessions",
		body:    `{"widget_id":"` + s4ID3 + `","page_url":"https://example.com/p","referrer":"https://example.com/r"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().WebchatSessionCreate(gomock.Any(), a, widget, "https://example.com/p", "https://example.com/r")
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases,
		s4Case{
			name: "PostWebchatSessions/invalid_widget_id", method: http.MethodPost, path: "/webchat_sessions",
			body:          `{"widget_id":"nope"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_WIDGET_ID",
		},
		s4Case{
			name: "PostWebchatSessions/optional_fields_empty", method: http.MethodPost, path: "/webchat_sessions",
			body: `{"widget_id":"` + s4ID3 + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatSessionCreate(gomock.Any(), a, widget, "", "").Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
	)
	s4RunCases(t, cases)
}

func Test_Ais_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetAis", path: "/ais",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().AIGetsByCustomerID(gomock.Any(), a, size, token)
		},
		mk: s4MkList[amai.WebhookMessage],
	})...)
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetAisIdParticipants", path: "/ais/" + s4ID1 + "/participants",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().AIParticipantGets(gomock.Any(), a, s4UUID(s4ID1), token, size)
		},
		mk: s4MkList[amparticipant.WebhookMessage],
	})...)

	ret := s4Obj[amai.WebhookMessage](s4ID1, nil)
	bad := "/ais/not-a-uuid"
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetAisId", method: http.MethodGet, path: "/ais/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIGet(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteAisId", method: http.MethodDelete, path: "/ais/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIDelete(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostAisIdActivateInsight", method: http.MethodPost, path: "/ais/" + s4ID1 + "/activate_insight",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIActivateInsight(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostAisIdDirectHashRegenerate", method: http.MethodPost, path: "/ais/" + s4ID1 + "/direct-hash-regenerate",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIDirectHashRegenerate(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: ret, okStatus: 200,
	})...)

	fullBody := `{"name":"ai-1","detail":"ai detail","type":"normal","engine_model":"model-x","parameter":{"temp":0.5},` +
		`"engine_key":"ek","engine_base_url":"https://example.com/v1","engine_base_key":"bk","rag_id":"` + s4ID2 + `",` +
		`"init_prompt":"hello","tts_type":"tts-a","tts_voice_id":"voice-1","stt_type":"stt-a","stt_language":"en-US",` +
		`"tool_names":["tool_a","tool_b"],"mcp_server_ids":["` + s4ID3 + `"],"auto_aicall_audit_enabled":true}`
	bk := "bk"
	mcpIDs := []uuid.UUID{s4UUID(s4ID3)}
	emptyMCP := []uuid.UUID{}
	tools := []amtool.ToolName{"tool_a", "tool_b"}
	param := map[string]any{"temp": 0.5}

	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostAis", method: http.MethodPost, path: "/ais", body: fullBody, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AICreate(gomock.Any(), a, "ai-1", "ai detail", amai.Type("normal"), amai.EngineModel("model-x"), param,
				"ek", "https://example.com/v1", &bk, s4UUID(s4ID2), "hello", amai.TTSType("tts-a"), "voice-1",
				amai.STTType("stt-a"), "en-US", tools, &mcpIDs, true)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases,
		s4Case{
			name: "PostAis/minimal_body_defaults", method: http.MethodPost, path: "/ais",
			body: `{"name":"min"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AICreate(gomock.Any(), a, "min", "", amai.Type(""), amai.EngineModel(""), map[string]any(nil),
					"", "", (*string)(nil), uuid.Nil, "", amai.TTSType(""), "",
					amai.STTType(""), "", []amtool.ToolName(nil), (*[]uuid.UUID)(nil), false).Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
		s4Case{
			name: "PostAis/explicit_empty_mcp_server_ids_is_non_nil_empty", method: http.MethodPost, path: "/ais",
			body: `{"name":"min","mcp_server_ids":[]}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AICreate(gomock.Any(), a, "min", "", amai.Type(""), amai.EngineModel(""), map[string]any(nil),
					"", "", (*string)(nil), uuid.Nil, "", amai.TTSType(""), "",
					amai.STTType(""), "", []amtool.ToolName(nil), &emptyMCP, false).Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
	)

	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutAisId", method: http.MethodPut, path: "/ais/" + s4ID1, body: fullBody, badJSON: true,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIUpdate(gomock.Any(), a, s4UUID(s4ID1), "ai-1", "ai detail", amai.Type("normal"), amai.EngineModel("model-x"), param,
				"ek", "https://example.com/v1", &bk, s4UUID(s4ID2), "hello", amai.TTSType("tts-a"), "voice-1",
				amai.STTType("stt-a"), "en-US", tools, &mcpIDs, true)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutAisId/minimal_body_defaults", method: http.MethodPut, path: "/ais/" + s4ID1,
		body: `{"name":"min"}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIUpdate(gomock.Any(), a, s4UUID(s4ID1), "min", "", amai.Type(""), amai.EngineModel(""), map[string]any(nil),
				"", "", (*string)(nil), uuid.Nil, "", amai.TTSType(""), "",
				amai.STTType(""), "", []amtool.ToolName(nil), (*[]uuid.UUID)(nil), false).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})

	// prompt histories
	histRet := s4Obj[amaiprompthistory.AIPromptHistory](s4ID2, nil)
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetAisIdPromptHistories", path: "/ais/" + s4ID1 + "/prompt_histories",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().AIPromptHistoryGetsByAIID(gomock.Any(), a, s4UUID(s4ID1), size, token)
		},
		mk: s4MkList[amaiprompthistory.AIPromptHistory],
	})...)
	cases = append(cases, s4Case{
		name: "GetAisIdPromptHistories/invalid_ai_id", method: http.MethodGet, path: "/ais/not-a-uuid/prompt_histories",
		wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID",
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetAisIdPromptHistoriesHistoryId", method: http.MethodGet, path: "/ais/" + s4ID1 + "/prompt_histories/" + s4ID2,
		badIDPath: "/ais/not-a-uuid/prompt_histories/" + s4ID2, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIPromptHistoryGet(gomock.Any(), a, s4UUID(s4ID1), s4UUID(s4ID2))
		},
		ret: histRet, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "GetAisIdPromptHistoriesHistoryId/invalid_history_id", method: http.MethodGet, path: "/ais/" + s4ID1 + "/prompt_histories/not-a-uuid",
		wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_HISTORY_ID",
	})
	s4RunCases(t, cases)
}
