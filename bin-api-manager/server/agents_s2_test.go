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

func Test_AgentsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetAgents unauthenticated", method: http.MethodGet, path: "/agents", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostAgents unauthenticated", method: http.MethodPost, path: "/agents", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostAgents bad json", method: http.MethodPost, path: "/agents", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetAgentsId unauthenticated", method: http.MethodGet, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAgentsId nil id", method: http.MethodGet, path: "/agents/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetAgentsId service error", method: http.MethodGet, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AgentGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "DeleteAgentsId unauthenticated", method: http.MethodDelete, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteAgentsId nil id", method: http.MethodDelete, path: "/agents/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "DeleteAgentsId service error", method: http.MethodDelete, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AgentDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PutAgentsId unauthenticated", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAgentsId nil id", method: http.MethodPut, path: "/agents/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAgentsId bad json", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutAgentsIdAddresses unauthenticated", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/addresses", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAgentsIdAddresses nil id", method: http.MethodPut, path: "/agents/00000000-0000-0000-0000-000000000000/addresses", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAgentsIdAddresses bad json", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/addresses", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutAgentsIdTagIds unauthenticated", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/tag_ids", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAgentsIdTagIds nil id", method: http.MethodPut, path: "/agents/00000000-0000-0000-0000-000000000000/tag_ids", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAgentsIdTagIds bad json", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/tag_ids", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutAgentsIdStatus unauthenticated", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/status", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAgentsIdStatus nil id", method: http.MethodPut, path: "/agents/00000000-0000-0000-0000-000000000000/status", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAgentsIdStatus bad json", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/status", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutAgentsIdPermission unauthenticated", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/permission", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAgentsIdPermission nil id", method: http.MethodPut, path: "/agents/00000000-0000-0000-0000-000000000000/permission", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAgentsIdPermission bad json", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/permission", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutAgentsIdPassword unauthenticated", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/password", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAgentsIdPassword nil id", method: http.MethodPut, path: "/agents/00000000-0000-0000-0000-000000000000/password", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAgentsIdPassword bad json", method: http.MethodPut, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/password", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostAgentsIdDirectHashRegenerate unauthenticated", method: http.MethodPost, path: "/agents/3bc539bc-c68b-11ec-b41f-0776699e7467/direct-hash-regenerate", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
