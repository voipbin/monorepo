package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const s1TranscribeID = "cd7f6294-0008-11f0-9b8a-afc07c8d9e08"

func Test_PostTranscribes_Errors(t *testing.T) {
	startBody := func(extra string) string {
		return `{"reference_type":"call","reference_id":"` + s1RefID + `","language":"en-US"` + extra + `}`
	}
	expectStart := func(id uuid.UUID, dir tmtranscribe.Direction, onEnd uuid.UUID, prov tmtranscribe.Provider) func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		return func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().TranscribeStart(gomock.Any(), a, id, "call", s1ID(s1RefID), "en-US", dir, onEnd, prov).Return(nil, err)
		}
	}

	s1Run(t, []s1Case{
		s1Unauth("POST", "/transcribes", startBody("")),
		s1BadJSON("POST", "/transcribes"),
		s1PermDenied("POST", "/transcribes", startBody(""), expectStart(uuid.Nil, tmtranscribe.DirectionBoth, uuid.Nil, "")),
		s1PermDenied("POST", "/transcribes", startBody(`,"direction":"in"`), expectStart(uuid.Nil, tmtranscribe.DirectionIn, uuid.Nil, "")),
		s1PermDenied("POST", "/transcribes", startBody(`,"direction":"sideways"`), expectStart(uuid.Nil, tmtranscribe.DirectionBoth, uuid.Nil, "")),
		s1PermDenied("POST", "/transcribes", startBody(`,"direction":""`), expectStart(uuid.Nil, tmtranscribe.DirectionBoth, uuid.Nil, "")),
		s1PermDenied("POST", "/transcribes",
			startBody(`,"id":"`+s1TranscribeID+`","on_end_flow_id":"`+s1FlowID+`","provider":"gcp"`),
			expectStart(s1ID(s1TranscribeID), tmtranscribe.DirectionBoth, s1ID(s1FlowID), tmtranscribe.Provider("gcp"))),
	})
}

func Test_GetTranscribes_Errors(t *testing.T) {
	cases := s1ListCases("/transcribes", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().TranscribeList(gomock.Any(), a, size, token, "", uuid.Nil).Return(nil, err)
	})
	cases = append(cases,
		s1Case{
			name:       "reference_type without reference_id is rejected",
			method:     "GET",
			path:       "/transcribes?reference_type=call",
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_REFERENCE_FILTER",
		},
		s1Case{
			name:       "reference_id without reference_type is rejected",
			method:     "GET",
			path:       "/transcribes?reference_id=" + s1RefID,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_REFERENCE_FILTER",
		},
		s1Case{
			name:   "reference filter pair is forwarded",
			method: "GET",
			path:   "/transcribes?reference_type=recording&reference_id=" + s1RefID,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TranscribeList(gomock.Any(), a, uint64(100), "", "recording", s1ID(s1RefID)).Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
	)
	s1Run(t, cases)
}

func Test_GetTranscribesId_Errors(t *testing.T) {
	path := "/transcribes/" + s1TranscribeID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TranscribeGet(gomock.Any(), a, s1ID(s1TranscribeID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/transcribes/not-a-uuid", ""),
		s1PermDenied("GET", path, "", expect),
	})
}

func Test_DeleteTranscribesId_Errors(t *testing.T) {
	path := "/transcribes/" + s1TranscribeID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TranscribeDelete(gomock.Any(), a, s1ID(s1TranscribeID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/transcribes/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", expect),
	})
}

func Test_PostTranscribesIdStop_Errors(t *testing.T) {
	path := "/transcribes/" + s1TranscribeID + "/stop"
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TranscribeStop(gomock.Any(), a, s1ID(s1TranscribeID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("POST", path, ""),
		s1BadID("POST", "/transcribes/not-a-uuid/stop", ""),
		s1PermDenied("POST", path, "", expect),
	})
}
