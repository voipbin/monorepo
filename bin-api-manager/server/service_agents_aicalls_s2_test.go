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

func Test_ServiceAgentsAicallsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetServiceAgentsAicalls unauthenticated", method: http.MethodGet, path: "/service_agents/aicalls", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsAicalls unauthenticated", method: http.MethodPost, path: "/service_agents/aicalls", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsAicalls bad json", method: http.MethodPost, path: "/service_agents/aicalls", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostServiceAgentsAicallsIdListen unauthenticated", method: http.MethodPost, path: "/service_agents/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467/listen", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsAicallsIdListen service error", method: http.MethodPost, path: "/service_agents/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467/listen", body: `{}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentAIcallListen(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
	}

	s2RunCases(t, tests)
}
