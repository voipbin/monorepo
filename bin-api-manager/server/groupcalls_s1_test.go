package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const s1GroupcallID = "dc80a355-0019-11f0-9699-b0d12d3e9469"

func Test_PostGroupcalls_Errors(t *testing.T) {
	body := `{"source":{"type":"tel","target":"+15555550100"},"destinations":[{"type":"tel","target":"+15555550101"}],"flow_id":"` + s1FlowID + `","ring_method":"ring_all","answer_method":"hangup_others"}`
	res := &cmgroupcall.WebhookMessage{Identity: commonidentity.Identity{ID: s1ID(s1GroupcallID)}}
	s1Run(t, []s1Case{
		s1Unauth("POST", "/groupcalls", body),
		s1BadJSON("POST", "/groupcalls"),
		s1PermDenied("POST", "/groupcalls", body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().GroupcallCreate(gomock.Any(), a,
				commonaddress.Address{Type: commonaddress.Type("tel"), Target: "+15555550100"},
				[]commonaddress.Address{{Type: commonaddress.Type("tel"), Target: "+15555550101"}},
				s1ID(s1FlowID), []fmaction.Action{}, cmgroupcall.RingMethod("ring_all"), cmgroupcall.AnswerMethod("hangup_others")).Return(nil, err)
		}),
		{
			name:   "minimal request uses zero values and returns the groupcall",
			method: "POST",
			path:   "/groupcalls",
			body:   `{"source":{},"destinations":[],"ring_method":"","answer_method":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().GroupcallCreate(gomock.Any(), a, commonaddress.Address{}, []commonaddress.Address{}, uuid.Nil, []fmaction.Action{},
					cmgroupcall.RingMethod(""), cmgroupcall.AnswerMethod("")).Return(res, nil)
			},
			wantBody: s1JSON(t, res),
		},
	})
}

func Test_GetGroupcalls_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/groupcalls", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().GroupcallList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_GroupcallsId_Errors(t *testing.T) {
	path := "/groupcalls/" + s1GroupcallID
	bad := "/groupcalls/not-a-uuid"
	id := s1ID(s1GroupcallID)
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", bad, ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().GroupcallGet(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("POST", path+"/hangup", ""),
		s1BadID("POST", bad+"/hangup", ""),
		s1PermDenied("POST", path+"/hangup", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().GroupcallHangup(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", bad, ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().GroupcallDelete(gomock.Any(), a, id).Return(nil, err)
		}),
	})
}
