package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"go.uber.org/mock/gomock"
)

func Test_GetQueuecalls_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/queuecalls?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/queuecalls?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallList(gomock.Any(), a, uint64(20), "tok").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/queuecalls?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallList(gomock.Any(), a, uint64(100), "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/queuecalls?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallList(gomock.Any(), a, uint64(20), "tok").Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_GetQueuecallsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/queuecalls/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallGet(gomock.Any(), a, s3UUID1).Return(&qmqueuecall.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueuecall.WebhookMessage{})},
	})
}

func Test_DeleteQueuecallsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/queuecalls/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallDelete(gomock.Any(), a, s3UUID1).Return(&qmqueuecall.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueuecall.WebhookMessage{})},
	})
}

func Test_PostQueuecallsIdKick_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1/kick", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/queuecalls/00000000-0000-0000-0000-000000000000/kick", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1/kick", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallKick(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/queuecalls/00000000-0000-4000-8000-0000000000a1/kick", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallKick(gomock.Any(), a, s3UUID1).Return(&qmqueuecall.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueuecall.WebhookMessage{})},
	})
}

func Test_PostQueuecallsReferenceIdIdKick_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/queuecalls/reference_id/00000000-0000-4000-8000-0000000000a1/kick", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/queuecalls/reference_id/00000000-0000-0000-0000-000000000000/kick", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/queuecalls/reference_id/00000000-0000-4000-8000-0000000000a1/kick", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallKickByReferenceID(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/queuecalls/reference_id/00000000-0000-4000-8000-0000000000a1/kick", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueuecallKickByReferenceID(gomock.Any(), a, s3UUID1).Return(&qmqueuecall.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueuecall.WebhookMessage{})},
	})
}
