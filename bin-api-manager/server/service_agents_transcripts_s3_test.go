package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsTranscripts_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/service_agents/transcripts?transcribe_id=00000000-0000-4000-8000-0000000000a2&page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/service_agents/transcripts?transcribe_id=00000000-0000-4000-8000-0000000000a2&page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentTranscriptList(gomock.Any(), a, uint64(20), "tok", s3UUID2).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/service_agents/transcripts?transcribe_id=00000000-0000-4000-8000-0000000000a2&page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentTranscriptList(gomock.Any(), a, uint64(100), "", s3UUID2).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/service_agents/transcripts?transcribe_id=00000000-0000-4000-8000-0000000000a2&page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentTranscriptList(gomock.Any(), a, uint64(20), "tok", s3UUID2).Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}
