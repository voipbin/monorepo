package server

import (
	"testing"

	ammessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"go.uber.org/mock/gomock"
)

const s1AicallID = "ed91b466-001a-11f0-87aa-c1e23e4fa57a"

func Test_GetServiceAgentsAimessages_Errors(t *testing.T) {
	base := "/service_agents/aimessages"
	q := "?aicall_id=" + s1AicallID
	s1Run(t, []s1Case{
		s1Unauth("GET", base+q, ""),
		{
			name:       "nil aicall_id",
			method:     "GET",
			path:       base + "?aicall_id=" + s1NilUUID,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_ID",
		},
		s1PermDenied("GET", base+q+"&page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentAImessageList(gomock.Any(), a, s1ID(s1AicallID), uint64(100), "tok").Return(nil, err)
		}),
		{
			name:   "empty result",
			method: "GET",
			path:   base + q + "&page_size=0",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentAImessageList(gomock.Any(), a, s1ID(s1AicallID), uint64(100), "").Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
	})
}

func Test_PostServiceAgentsAimessages_Errors(t *testing.T) {
	path := "/service_agents/aimessages"
	body := `{"aicall_id":"` + s1AicallID + `","role":"user","content":"hello"}`
	s1Run(t, []s1Case{
		s1Unauth("POST", path, body),
		s1BadJSON("POST", path),
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentAImessageCreate(gomock.Any(), a, s1ID(s1AicallID), ammessage.Role("user"), "hello").Return(nil, err)
		}),
	})
}
