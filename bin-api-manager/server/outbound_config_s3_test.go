package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	cerrors "monorepo/bin-common-handler/models/errors"

	"go.uber.org/mock/gomock"
)

func Test_GetOutboundConfig_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/outbound_config", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/outbound_config", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().OutboundConfigSelfGet(gomock.Any(), a).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/outbound_config", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().OutboundConfigSelfGet(gomock.Any(), a).Return(&cmoutboundconfig.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmoutboundconfig.WebhookMessage{})},
	})
}

func Test_PutOutboundConfig_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/outbound_config", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "PUT", path: "/outbound_config", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}
