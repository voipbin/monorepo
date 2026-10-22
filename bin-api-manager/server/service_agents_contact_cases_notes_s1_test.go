package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"go.uber.org/mock/gomock"
)

const (
	s1CaseID = "fea2c577-001b-11f0-98bb-d2f34f50b68b"
	s1NoteID = "0fb3d688-001c-11f0-89cc-e3045061c79c"
)

func Test_ServiceAgentsContactCasesIdNotes_Errors(t *testing.T) {
	path := "/service_agents/contact_cases/" + s1CaseID + "/notes"
	id := s1ID(s1CaseID)
	body := `{"text":"note text"}`
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1PermDenied("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentCaseNoteList(gomock.Any(), a, id).Return(nil, err)
		}),
		{
			name:   "list success with no notes",
			method: "GET",
			path:   path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentCaseNoteList(gomock.Any(), a, id).Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
		s1Unauth("POST", path, body),
		s1BadJSON("POST", path),
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentCaseNoteCreate(gomock.Any(), a, id, "note text").Return(nil, err)
		}),
		s1Unauth("DELETE", path+"/"+s1NoteID, ""),
		s1PermDenied("DELETE", path+"/"+s1NoteID, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentCaseNoteDelete(gomock.Any(), a, id, s1ID(s1NoteID)).Return(err)
		}),
		{
			name:   "delete success returns an empty object",
			method: "DELETE",
			path:   path + "/" + s1NoteID,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentCaseNoteDelete(gomock.Any(), a, id, s1ID(s1NoteID)).Return(nil)
			},
			wantBody: `{}`,
		},
	})
}
