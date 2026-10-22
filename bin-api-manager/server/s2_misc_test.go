package server

import (
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetMe_NonAgentAndServiceError(t *testing.T) {
	tests := []s2Case{
		{
			name:       "GetMe accesskey identity is rejected",
			method:     http.MethodGet,
			path:       "/me",
			identity:   auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{}),
			wantStatus: cerrors.StatusPermissionDenied,
			wantReason: "AGENT_AUTH_REQUIRED",
		},
		{
			name:       "GetMe service error",
			method:     http.MethodGet,
			path:       "/me",
			wantStatus: cerrors.StatusPermissionDenied,
			wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AgentGet(gomock.Any(), a, uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")).Return(nil, s2Err)
			},
		},
	}

	s2RunCases(t, tests)
}

func Test_AuthPasswordStubs_RouteNotFound(t *testing.T) {
	tests := []s2Case{
		{
			name:       "PostAuthPasswordForgot",
			method:     http.MethodPost,
			path:       "/auth/password-forgot",
			body:       `{}`,
			wantStatus: cerrors.StatusNotFound,
			wantReason: "ROUTE_NOT_FOUND",
		},
		{
			name:       "GetAuthPasswordReset",
			method:     http.MethodGet,
			path:       "/auth/password-reset?token=0000000000000000000000000000000000000000000000000000000000000000",
			wantStatus: cerrors.StatusNotFound,
			wantReason: "ROUTE_NOT_FOUND",
		},
		{
			name:       "PostAuthPasswordReset",
			method:     http.MethodPost,
			path:       "/auth/password-reset",
			body:       `{}`,
			wantStatus: cerrors.StatusNotFound,
			wantReason: "ROUTE_NOT_FOUND",
		},
	}

	s2RunCases(t, tests)
}

func Test_PostTransfers_ServiceError(t *testing.T) {
	tests := []s2Case{
		{
			name:       "PostTransfers service error carries converted arguments",
			method:     http.MethodPost,
			path:       "/transfers",
			body:       `{"transfer_type":"attended","transferer_call_id":"3bc539bc-c68b-11ec-b41f-0776699e7467","transferee_addresses":[]}`,
			wantStatus: cerrors.StatusPermissionDenied,
			wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TransferStart(gomock.Any(), a, tmtransfer.Type("attended"), uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"), gomock.Len(0)).Return(nil, s2Err)
			},
		},
	}

	s2RunCases(t, tests)
}
