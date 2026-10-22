package server

import (
	"strings"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const s1ActiveflowID = "bc6e5183-0007-11f0-8a7f-9ebf6b7c8d07"

func Test_GetActiveflows_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/activeflows", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().ActiveflowList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_PostActiveflows_Errors(t *testing.T) {
	tooMany := make([]string, 0, 101)
	for i := 0; i < 101; i++ {
		tooMany = append(tooMany, `"k`+strings.Repeat("x", i%7)+string(rune('a'+i%26))+string(rune('a'+i/26))+`":"v"`)
	}
	tooManyBody := `{"variables":{` + strings.Join(tooMany, ",") + `}}`
	bigValueBody := `{"variables":{"k":"` + strings.Repeat("v", 32*1024+1) + `"}}`

	s1Run(t, []s1Case{
		s1BadJSON("POST", "/activeflows"),
		{
			name:       "more than 100 variables",
			method:     "POST",
			path:       "/activeflows",
			body:       tooManyBody,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_VARIABLES",
		},
		{
			name:       "variable value above 32KB",
			method:     "POST",
			path:       "/activeflows",
			body:       bigValueBody,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_VARIABLES",
		},
		{
			name:       "webhook uri with a non-http scheme",
			method:     "POST",
			path:       "/activeflows",
			body:       `{"webhook_uri":"ftp://example.com/hook"}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_WEBHOOK_URI",
		},
		{
			name:       "webhook uri without host",
			method:     "POST",
			path:       "/activeflows",
			body:       `{"webhook_uri":"https://"}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_WEBHOOK_URI",
		},
		{
			name:       "unknown webhook method",
			method:     "POST",
			path:       "/activeflows",
			body:       `{"webhook_uri":"https://example.com/hook","webhook_method":"PATCH"}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_WEBHOOK_METHOD",
		},
		s1PermDenied("POST", "/activeflows",
			`{"id":"`+s1ActiveflowID+`","flow_id":"`+s1FlowID+`","variables":{"a":"b"},"webhook_uri":"https://example.com/hook","webhook_method":"POST"}`,
			func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
				m.EXPECT().ActiveflowCreate(gomock.Any(), a, s1ID(s1ActiveflowID), s1ID(s1FlowID), []fmaction.Action{}, map[string]string{"a": "b"}, "https://example.com/hook", fmactiveflow.WebhookMethodPost).Return(nil, err)
			}),
		s1PermDenied("POST", "/activeflows", `{}`,
			func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
				m.EXPECT().ActiveflowCreate(gomock.Any(), a, uuid.Nil, uuid.Nil, []fmaction.Action{}, map[string]string(nil), "", fmactiveflow.WebhookMethodNone).Return(nil, err)
			}),
	})
}

func Test_GetActiveflowsId_Errors(t *testing.T) {
	path := "/activeflows/" + s1ActiveflowID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().ActiveflowGet(gomock.Any(), a, s1ID(s1ActiveflowID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1PermDenied("GET", path, "", expect),
		s1NotFound("GET", path, "", expect),
	})
}

func Test_DeleteActiveflowsId_Errors(t *testing.T) {
	path := "/activeflows/" + s1ActiveflowID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().ActiveflowDelete(gomock.Any(), a, s1ID(s1ActiveflowID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/activeflows/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", expect),
	})
}
