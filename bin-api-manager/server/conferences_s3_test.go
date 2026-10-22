package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"go.uber.org/mock/gomock"
)

func Test_GetConferences_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/conferences?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/conferences?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceList(gomock.Any(), a, uint64(20), "tok").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/conferences?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceList(gomock.Any(), a, uint64(100), "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/conferences?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceList(gomock.Any(), a, uint64(20), "tok").Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_PostConferences_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/conferences", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/conferences", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_GetConferencesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/conferences/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/conferences/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/conferences/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/conferences/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceGet(gomock.Any(), a, s3UUID1).Return(&cfconference.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cfconference.WebhookMessage{})},
	})
}

func Test_PutConferencesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/conferences/00000000-0000-4000-8000-0000000000a1", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/conferences/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/conferences/00000000-0000-4000-8000-0000000000a1", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "invalid_pre_flow_id", method: "PUT", path: "/conferences/00000000-0000-4000-8000-0000000000a1", body: `{"pre_flow_id":"x"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_PRE_FLOW_ID"},
		{name: "invalid_post_flow_id", method: "PUT", path: "/conferences/00000000-0000-4000-8000-0000000000a1", body: `{"post_flow_id":"x"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_POST_FLOW_ID"},
	})
}

func Test_DeleteConferencesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/conferences/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/conferences/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/conferences/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/conferences/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceDelete(gomock.Any(), a, s3UUID1).Return(&cfconference.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cfconference.WebhookMessage{})},
	})
}

func Test_PostConferencesIdRecordingStart_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/recording_start", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/conferences/00000000-0000-0000-0000-000000000000/recording_start", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/recording_start", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_PostConferencesIdRecordingStop_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/recording_stop", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/conferences/00000000-0000-0000-0000-000000000000/recording_stop", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/recording_stop", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceRecordingStop(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/recording_stop", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceRecordingStop(gomock.Any(), a, s3UUID1).Return(&cfconference.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cfconference.WebhookMessage{})},
	})
}

func Test_PostConferencesIdTranscribeStart_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/transcribe_start", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/conferences/00000000-0000-0000-0000-000000000000/transcribe_start", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/transcribe_start", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_PostConferencesIdTranscribeStop_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/transcribe_stop", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/conferences/00000000-0000-0000-0000-000000000000/transcribe_stop", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/transcribe_stop", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceTranscribeStop(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/transcribe_stop", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceTranscribeStop(gomock.Any(), a, s3UUID1).Return(&cfconference.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cfconference.WebhookMessage{})},
	})
}

func Test_GetConferencesIdMediaStream_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/conferences/00000000-0000-4000-8000-0000000000a1/media_stream?encapsulation=audiosocket", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/conferences/00000000-0000-0000-0000-000000000000/media_stream?encapsulation=audiosocket", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
	})
}

func Test_PostConferencesIdDirectHashRegenerate_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/conferences/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConferenceDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(&cfconference.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cfconference.WebhookMessage{})},
	})
}
