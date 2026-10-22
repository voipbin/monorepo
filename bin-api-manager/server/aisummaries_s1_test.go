package server

import (
	"testing"

	amsummary "monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	s1SummaryID = "8f3b2e50-0004-11f0-9d4c-6b8c3e4f5a04"
	s1RefID     = "9a4c3f61-0005-11f0-8e5d-7c9d4f5a6b05"
	s1FlowID    = "ab5d4072-0006-11f0-9f6e-8dae5a6b7c06"
)

func Test_PostAisummaries_Errors(t *testing.T) {
	body := `{"on_end_flow_id":"` + s1FlowID + `","reference_type":"call","reference_id":"` + s1RefID + `","language":"en-US"}`
	s1Run(t, []s1Case{
		s1Unauth("POST", "/aisummaries", body),
		s1BadJSON("POST", "/aisummaries"),
		s1PermDenied("POST", "/aisummaries", body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().AISummaryCreate(gomock.Any(), a, s1ID(s1FlowID), amsummary.ReferenceTypeCall, s1ID(s1RefID), "en-US").Return(nil, err)
		}),
	})
}

func Test_GetAisummaries_Errors(t *testing.T) {
	cases := s1ListCases("/aisummaries", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().AISummaryList(gomock.Any(), a, size, token, "", uuid.Nil).Return(nil, err)
	})
	cases = append(cases,
		s1Case{
			name:       "reference_type without reference_id is rejected",
			method:     "GET",
			path:       "/aisummaries?reference_type=call",
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_REFERENCE_FILTER",
		},
		s1Case{
			name:       "reference_id without reference_type is rejected",
			method:     "GET",
			path:       "/aisummaries?reference_id=" + s1RefID,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_REFERENCE_FILTER",
		},
		s1Case{
			name:   "reference filter pair is forwarded",
			method: "GET",
			path:   "/aisummaries?reference_type=call&reference_id=" + s1RefID,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AISummaryList(gomock.Any(), a, uint64(100), "", "call", s1ID(s1RefID)).Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
	)
	s1Run(t, cases)
}

func Test_GetAisummariesId_Errors(t *testing.T) {
	path := "/aisummaries/" + s1SummaryID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().AISummaryGet(gomock.Any(), a, s1ID(s1SummaryID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/aisummaries/not-a-uuid", ""),
		s1PermDenied("GET", path, "", expect),
	})
}

func Test_DeleteAisummariesId_Errors(t *testing.T) {
	path := "/aisummaries/" + s1SummaryID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().AISummaryDelete(gomock.Any(), a, s1ID(s1SummaryID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/aisummaries/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", expect),
	})
}

func Test_PostAisummariesIdRegenerate(t *testing.T) {
	path := "/aisummaries/" + s1SummaryID + "/regenerate"
	res := &amsummary.WebhookMessage{Identity: commonidentity.Identity{ID: s1ID(s1SummaryID)}}
	s1Run(t, []s1Case{
		s1Unauth("POST", path, ""),
		s1BadID("POST", "/aisummaries/00000000-0000-0000-0000-000000000000/regenerate", ""),
		s1PermDenied("POST", path, `{"language":"ko-KR"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().AISummaryRegenerate(gomock.Any(), a, s1ID(s1SummaryID), "ko-KR").Return(nil, err)
		}),
		{
			name:   "invalid body is treated as no language re-selection",
			method: "POST",
			path:   path,
			body:   `{"language":`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AISummaryRegenerate(gomock.Any(), a, s1ID(s1SummaryID), "").Return(res, nil)
			},
			wantBody: s1JSON(t, res),
		},
		{
			name:   "language from the body is forwarded",
			method: "POST",
			path:   path,
			body:   `{"language":"ko-KR"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AISummaryRegenerate(gomock.Any(), a, s1ID(s1SummaryID), "ko-KR").Return(res, nil)
			},
			wantBody: s1JSON(t, res),
		},
	})
}
