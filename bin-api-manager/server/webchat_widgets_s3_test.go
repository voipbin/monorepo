package server

import (
	"testing"

	uuid "github.com/gofrs/uuid"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"go.uber.org/mock/gomock"
)

func Test_GetWebchatWidgets_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/webchat_widgets?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/webchat_widgets?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetList(gomock.Any(), a, uint64(20), "tok").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/webchat_widgets?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetList(gomock.Any(), a, uint64(100), "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/webchat_widgets?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetList(gomock.Any(), a, uint64(20), "tok").Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_PostWebchatWidgets_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/webchat_widgets", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/webchat_widgets", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "invalid_theme_color", method: "POST", path: "/webchat_widgets", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1","theme_config":{"header_background_color":"red"}}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_THEME_CONFIG"},
		{name: "service_error", method: "POST", path: "/webchat_widgets", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetCreate(gomock.Any(), a, "n", uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000b1"), uuid.Nil, 0, nil).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_DeleteWebchatWidgetsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/webchat_widgets/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetDelete(gomock.Any(), a, s3UUID1).Return(&wcwidget.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&wcwidget.WebhookMessage{})},
	})
}

func Test_GetWebchatWidgetsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/webchat_widgets/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetGet(gomock.Any(), a, s3UUID1).Return(&wcwidget.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&wcwidget.WebhookMessage{})},
	})
}

func Test_PutWebchatWidgetsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/webchat_widgets/00000000-0000-0000-0000-000000000000", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "invalid_theme_color", method: "PUT", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1","theme_config":{"header_background_color":"red"}}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_THEME_CONFIG"},
		{name: "service_error", method: "PUT", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1", body: `{"name":"n","session_flow_id":"00000000-0000-4000-8000-0000000000b1"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetUpdate(gomock.Any(), a, s3UUID1, "n", uuid.FromStringOrNil("00000000-0000-4000-8000-0000000000b1"), uuid.Nil, 0, nil).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_PostWebchatWidgetsIdDirectHashRegenerate_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1/direct_hash_regenerate", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/webchat_widgets/00000000-0000-0000-0000-000000000000/direct_hash_regenerate", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1/direct_hash_regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/webchat_widgets/00000000-0000-4000-8000-0000000000a1/direct_hash_regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatWidgetDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(&wcwidget.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&wcwidget.WebhookMessage{})},
	})
}
