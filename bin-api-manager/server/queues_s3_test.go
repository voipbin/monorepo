package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	qmqueue "monorepo/bin-queue-manager/models/queue"

	"go.uber.org/mock/gomock"
)

func Test_GetQueues_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/queues?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/queues?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueList(gomock.Any(), a, uint64(20), "tok").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/queues?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueList(gomock.Any(), a, uint64(100), "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/queues?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueList(gomock.Any(), a, uint64(20), "tok").Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_PostQueues_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/queues", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/queues", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_DeleteQueuesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/queues/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/queues/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/queues/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/queues/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueDelete(gomock.Any(), a, s3UUID1).Return(&qmqueue.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueue.WebhookMessage{})},
	})
}

func Test_GetQueuesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/queues/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/queues/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/queues/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/queues/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueGet(gomock.Any(), a, s3UUID1).Return(&qmqueue.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueue.WebhookMessage{})},
	})
}

func Test_PutQueuesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/queues/00000000-0000-4000-8000-0000000000a1", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/queues/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/queues/00000000-0000-4000-8000-0000000000a1", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_PutQueuesIdTagIds_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/queues/00000000-0000-4000-8000-0000000000a1/tag_ids", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/queues/00000000-0000-0000-0000-000000000000/tag_ids", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/queues/00000000-0000-4000-8000-0000000000a1/tag_ids", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_PutQueuesIdRoutingMethod_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/queues/00000000-0000-4000-8000-0000000000a1/routing_method", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/queues/00000000-0000-0000-0000-000000000000/routing_method", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/queues/00000000-0000-4000-8000-0000000000a1/routing_method", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_PostQueuesIdDirectHashRegenerate_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/queues/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "POST", path: "/queues/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/queues/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().QueueDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(&qmqueue.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&qmqueue.WebhookMessage{})},
	})
}
