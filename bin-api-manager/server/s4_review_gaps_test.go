package server

import (
	"net/http"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// Test_PutAgentsIdPassword_EmptyPasswordRejected verifies that a missing or empty password never reaches
// the service handler (no expectation is registered, so any call fails the test).
func Test_PutAgentsIdPassword_EmptyPasswordRejected(t *testing.T) {
	path := "/agents/" + s4ID1 + "/password"
	s4RunCases(t, []s4Case{
		{
			name: "missing_password", method: http.MethodPut, path: path, body: `{}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		{
			name: "empty_password", method: http.MethodPut, path: path, body: `{"password":""}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
	})
}

// Test_PutRagsId_FieldSelection verifies the empty update is rejected and that only the provided fields
// are forwarded as the update map.
func Test_PutRagsId_FieldSelection(t *testing.T) {
	path := "/rags/" + s4ID1
	s4RunCases(t, []s4Case{
		{
			name: "empty_body", method: http.MethodPut, path: path, body: `{}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		{
			name: "only_name_is_forwarded", method: http.MethodPut, path: path, body: `{"name":"n1"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagUpdate(gomock.Any(), a, s4UUID(s4ID1), map[rmrag.Field]any{rmrag.FieldName: "n1"}).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
		{
			name: "name_and_description_are_forwarded", method: http.MethodPut, path: path, body: `{"name":"n1","description":"d1"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RagUpdate(gomock.Any(), a, s4UUID(s4ID1), map[rmrag.Field]any{rmrag.FieldName: "n1", rmrag.FieldDescription: "d1"}).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
	})
}

// Test_PutWebchatWidgetsId_SessionIdleTimeout verifies a provided session_idle_timeout is forwarded.
func Test_PutWebchatWidgetsId_SessionIdleTimeout(t *testing.T) {
	flowID := "00000000-0000-4000-8000-0000000000b1"
	s4RunCases(t, []s4Case{
		{
			name: "idle_timeout_forwarded", method: http.MethodPut, path: "/webchat_widgets/" + s4ID1,
			body: `{"name":"n","session_flow_id":"` + flowID + `","session_idle_timeout":30}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatWidgetUpdate(gomock.Any(), a, s4UUID(s4ID1), "n", uuid.FromStringOrNil(flowID), uuid.Nil, 30, nil).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
	})
}

// Test_GetAgents_StatusNoneAddsNoFilter verifies an empty (none) status is not turned into a status filter, while a
// real status is.
func Test_GetAgents_StatusNoneAddsNoFilter(t *testing.T) {
	base := map[string]string{"customer_id": "5f7a1d3c-9e8b-4c2a-b6d4-3a1f0e9c8b77", "deleted": "false"}
	withStatus := map[string]string{"customer_id": base["customer_id"], "deleted": "false", "status": string(amagent.StatusAvailable)}
	s4RunCases(t, []s4Case{
		{
			name: "status_empty_means_none", method: http.MethodGet, path: "/agents?page_size=5&status=",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AgentList(gomock.Any(), a, uint64(5), "", base).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
		{
			name: "status_available", method: http.MethodGet, path: "/agents?page_size=5&status=available",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AgentList(gomock.Any(), a, uint64(5), "", withStatus).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
	})
}

// Test_PutConferencesId_FlowIDsForwarded verifies valid pre_flow_id and post_flow_id reach the service as
// parsed UUIDs, and omitted ones stay nil.
func Test_PutConferencesId_FlowIDsForwarded(t *testing.T) {
	path := "/conferences/" + s4ID1
	pre := uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000c1")
	post := uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000c2")
	name := "conf"
	s4RunCases(t, []s4Case{
		{
			name: "both_flow_ids", method: http.MethodPut, path: path,
			body: `{"name":"conf","pre_flow_id":"` + pre.String() + `","post_flow_id":"` + post.String() + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ConferenceUpdate(gomock.Any(), a, s4UUID(s4ID1), &name, nil, nil, nil, &pre, &post).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
		{
			name: "flow_ids_omitted", method: http.MethodPut, path: path, body: `{"name":"conf"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ConferenceUpdate(gomock.Any(), a, s4UUID(s4ID1), &name, nil, nil, nil, nil, nil).Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
	})
}
