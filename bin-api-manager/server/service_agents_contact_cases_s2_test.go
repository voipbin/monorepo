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

func Test_ServiceAgentsContactCasesHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetServiceAgentsContactCases unauthenticated", method: http.MethodGet, path: "/service_agents/contact_cases", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsContactCasesId unauthenticated", method: http.MethodGet, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsContactCasesId service error", method: http.MethodGet, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentCaseGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PostServiceAgentsContactCasesIdClose unauthenticated", method: http.MethodPost, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/close", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsContactCasesIdClose service error", method: http.MethodPost, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/close", body: `{}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentCaseClose(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PostServiceAgentsContactCasesIdAssign unauthenticated", method: http.MethodPost, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/assign", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsContactCasesIdAssign bad json", method: http.MethodPost, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/assign", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostServiceAgentsContactCasesIdUnassign unauthenticated", method: http.MethodPost, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/unassign", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsContactCasesIdUnassign service error", method: http.MethodPost, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467/unassign", body: `{}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentCaseUnassign(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PutServiceAgentsContactCasesId unauthenticated", method: http.MethodPut, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutServiceAgentsContactCasesId bad json", method: http.MethodPut, path: "/service_agents/contact_cases/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	}

	s2RunCases(t, tests)
}
