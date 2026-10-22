package server

import (
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const s1CallID = "ef91b4b6-000a-11f0-9daa-c1e29eafb00a"

// s1CallOp describes a per-call endpoint that returns only an error from the
// servicehandler (no response resource).
type s1CallOp struct {
	name   string
	method string
	suffix string
	body   string
	expect func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error)
}

func s1CallOps() []s1CallOp {
	id := s1ID(s1CallID)
	return []s1CallOp{
		{"hold on", "POST", "/hold", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallHoldOn(gomock.Any(), a, id).Return(err)
		}},
		{"hold off", "DELETE", "/hold", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallHoldOff(gomock.Any(), a, id).Return(err)
		}},
		{"moh on", "POST", "/moh", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallMOHOn(gomock.Any(), a, id).Return(err)
		}},
		{"moh off", "DELETE", "/moh", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallMOHOff(gomock.Any(), a, id).Return(err)
		}},
		{"silence on", "POST", "/silence", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallSilenceOn(gomock.Any(), a, id).Return(err)
		}},
		{"silence off", "DELETE", "/silence", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallSilenceOff(gomock.Any(), a, id).Return(err)
		}},
		{"mute on both", "POST", "/mute", `{"direction":"both"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallMuteOn(gomock.Any(), a, id, cmcall.MuteDirectionBoth).Return(err)
		}},
		{"mute on default direction", "POST", "/mute", `{}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallMuteOn(gomock.Any(), a, id, cmcall.MuteDirectionNone).Return(err)
		}},
		{"mute off in", "DELETE", "/mute", `{"direction":"in"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallMuteOff(gomock.Any(), a, id, cmcall.MuteDirection("in")).Return(err)
		}},
		{"talk", "POST", "/talk", `{"text":"hello","language":"en-US","provider":"gcp","voice_id":"v1"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallTalk(gomock.Any(), a, id, "hello", "en-US", "gcp", "v1").Return(err)
		}},
		{"talk with empty body object", "POST", "/talk", `{}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallTalk(gomock.Any(), a, id, "", "", "", "").Return(err)
		}},
	}
}

func Test_CallsIdOperations(t *testing.T) {
	var tests []s1Case
	for _, op := range s1CallOps() {
		path := "/calls/" + s1CallID + op.suffix
		opName := op.name

		unauth := s1Unauth(op.method, path, op.body)
		unauth.name = opName + ": no auth identity"

		badID := s1BadID(op.method, "/calls/not-a-uuid"+op.suffix, op.body)
		badID.name = opName + ": invalid id"

		denied := s1PermDenied(op.method, path, op.body, op.expect)
		denied.name = opName + ": permission denied from servicehandler"

		expect := op.expect
		ok := s1Case{
			name:   opName + ": success returns an empty 200",
			method: op.method,
			path:   path,
			body:   op.body,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				expect(m, a, nil)
			},
			wantBody: "",
		}
		tests = append(tests, unauth, badID, denied, ok)
	}

	// endpoints whose body is mandatory for BindJSON
	for _, spec := range []struct{ name, method, suffix string }{
		{"mute on", "POST", "/mute"},
		{"mute off", "DELETE", "/mute"},
		{"talk", "POST", "/talk"},
	} {
		bad := s1BadJSON(spec.method, "/calls/"+s1CallID+spec.suffix)
		bad.name = spec.name + ": invalid json body"
		tests = append(tests, bad)
	}
	s1Run(t, tests)
}

func Test_PostCalls_Errors(t *testing.T) {
	body := `{"flow_id":"` + s1FlowID + `","source":{"type":"tel","target":"+15555550100"},"destinations":[{"type":"tel","target":"+15555550101"}],"anonymous":"auto","variables":{"a":"b"}}`
	wantSource := commonaddress.Address{Type: commonaddress.Type("tel"), Target: "+15555550100"}
	res := []*cmcall.WebhookMessage{{Identity: commonidentity.Identity{ID: s1ID(s1CallID)}}}
	resGroup := []*cmgroupcall.WebhookMessage{{Identity: commonidentity.Identity{ID: s1ID(s1RefID)}}}

	s1Run(t, []s1Case{
		s1Unauth("POST", "/calls", body),
		s1BadJSON("POST", "/calls"),
		{
			name:       "more than 100 variables are rejected",
			method:     "POST",
			path:       "/calls",
			body:       `{"variables":{` + s1Vars(101) + `}}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_VARIABLES",
		},
		s1PermDenied("POST", "/calls", body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallCreate(gomock.Any(), a, s1ID(s1FlowID), []fmaction.Action{}, &wantSource,
				[]commonaddress.Address{{Type: commonaddress.Type("tel"), Target: "+15555550101"}}, "auto", map[string]string{"a": "b"}).Return(nil, nil, err)
		}),
		{
			name:   "empty request uses zero values",
			method: "POST",
			path:   "/calls",
			body:   `{}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().CallCreate(gomock.Any(), a, uuid.Nil, []fmaction.Action{}, &commonaddress.Address{}, []commonaddress.Address{}, "", map[string]string(nil)).Return(res, resGroup, nil)
			},
			wantBody: s1JSON(t, struct {
				Calls      []*cmcall.WebhookMessage      `json:"calls"`
				Groupcalls []*cmgroupcall.WebhookMessage `json:"groupcalls"`
			}{res, resGroup}),
		},
	})
}

func Test_GetCalls_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/calls", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().CallList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_CallsId_SingleResource(t *testing.T) {
	path := "/calls/" + s1CallID
	id := s1ID(s1CallID)
	res := &cmcall.WebhookMessage{Identity: commonidentity.Identity{ID: id}}

	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/calls/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallDelete(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/calls/not-a-uuid", ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallGet(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("POST", path+"/hangup", ""),
		s1BadID("POST", "/calls/not-a-uuid/hangup", ""),
		s1PermDenied("POST", path+"/hangup", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallHangup(gomock.Any(), a, id).Return(nil, err)
		}),
		{
			name:   "hangup success returns the call",
			method: "POST",
			path:   path + "/hangup",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().CallHangup(gomock.Any(), a, id).Return(res, nil)
			},
			wantBody: s1JSON(t, res),
		},
		s1Unauth("POST", path+"/recording_stop", ""),
		s1BadID("POST", "/calls/not-a-uuid/recording_stop", ""),
		s1PermDenied("POST", path+"/recording_stop", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallRecordingStop(gomock.Any(), a, id).Return(nil, err)
		}),
	})
}

func Test_PostCallsIdRecordingStart_Errors(t *testing.T) {
	path := "/calls/" + s1CallID + "/recording_start"
	body := `{"format":"wav","end_of_silence":5,"end_of_key":"#","duration":60,"on_end_flow_id":"` + s1FlowID + `"}`
	s1Run(t, []s1Case{
		s1Unauth("POST", path, body),
		s1BadID("POST", "/calls/not-a-uuid/recording_start", body),
		s1BadJSON("POST", path),
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallRecordingStart(gomock.Any(), a, s1ID(s1CallID), cmrecording.Format("wav"), 5, "#", 60, s1ID(s1FlowID)).Return(nil, err)
		}),
	})
}

func Test_GetCallsIdMediaStream_Errors(t *testing.T) {
	path := "/calls/" + s1CallID + "/media_stream"
	isWriter := gomock.Cond(func(x any) bool { _, ok := x.(http.ResponseWriter); return ok })
	isStreamReq := gomock.Cond(func(x any) bool {
		r, ok := x.(*http.Request)
		return ok && r.URL.Path == path
	})
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/calls/not-a-uuid/media_stream", ""),
		s1PermDenied("GET", path+"?encapsulation=rtp", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CallMediaStreamStart(gomock.Any(), a, s1ID(s1CallID), "rtp", isWriter, isStreamReq).Return(err)
		}),
	})
}
