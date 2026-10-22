package server

import (
	"testing"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"go.uber.org/mock/gomock"
)

const s1RecordingID = "de80a3a5-0009-11f0-8c9b-b0d18d9eaf09"

func Test_GetRecordings_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/recordings", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().RecordingList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_GetRecordingsId_Errors(t *testing.T) {
	path := "/recordings/" + s1RecordingID
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/recordings/not-a-uuid", ""),
		s1PermDenied("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().RecordingGet(gomock.Any(), a, s1ID(s1RecordingID)).Return(nil, err)
		}),
	})
}

func Test_GetRecordingsIdPlayfiles(t *testing.T) {
	path := "/recordings/" + s1RecordingID + "/playfiles"
	files := []openapi_server.ApiManagerRecordingPlayfile{}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/recordings/not-a-uuid/playfiles", ""),
		s1PermDenied("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().RecordingPlayfilesGet(gomock.Any(), a, s1ID(s1RecordingID)).Return(nil, err)
		}),
		{
			name:   "success returns the playfiles",
			method: "GET",
			path:   path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RecordingPlayfilesGet(gomock.Any(), a, s1ID(s1RecordingID)).Return(files, nil)
			},
			wantBody: `[]`,
		},
	})
}

func Test_DeleteRecordingsId_Errors(t *testing.T) {
	path := "/recordings/" + s1RecordingID
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/recordings/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().RecordingDelete(gomock.Any(), a, s1ID(s1RecordingID)).Return(nil, err)
		}),
	})
}

func Test_GetRecordingsIdTranscribes_Errors(t *testing.T) {
	path := "/recordings/" + s1RecordingID + "/transcribes"
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/recordings/not-a-uuid/transcribes", ""),
		s1PermDenied("GET", path+"?page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().RecordingTranscribeList(gomock.Any(), a, s1ID(s1RecordingID), uint64(100), "tok").Return(nil, err)
		}),
	})
}

func Test_GetRecordingsIdTranscripts_Errors(t *testing.T) {
	path := "/recordings/" + s1RecordingID + "/transcripts"
	withTranscribe := path + "?transcribe_id=" + s1TranscribeID
	s1Run(t, []s1Case{
		s1Unauth("GET", withTranscribe, ""),
		s1BadID("GET", "/recordings/not-a-uuid/transcripts?transcribe_id="+s1TranscribeID, ""),
		{
			name:       "invalid transcribe_id",
			method:     "GET",
			path:       path + "?transcribe_id=not-a-uuid",
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_TRANSCRIBE_ID",
		},
		s1PermDenied("GET", withTranscribe+"&page_size=0&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().RecordingTranscriptList(gomock.Any(), a, s1ID(s1RecordingID), s1ID(s1TranscribeID), uint64(100), "tok").Return(nil, err)
		}),
	})
}
