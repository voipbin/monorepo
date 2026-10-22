package server

import (
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentsAgentsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetServiceAgentsAgents unauthenticated", method: http.MethodGet, path: "/service_agents/agents", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsAgents service error", method: http.MethodGet, path: "/service_agents/agents?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentAgentList(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "GetServiceAgentsAgentsId unauthenticated", method: http.MethodGet, path: "/service_agents/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsAgentsId nil id", method: http.MethodGet, path: "/service_agents/agents/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetServiceAgentsAgentsId service error", method: http.MethodGet, path: "/service_agents/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentAgentGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
	}

	s2RunCases(t, tests)
}
