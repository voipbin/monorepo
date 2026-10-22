package server

import (
	"fmt"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtag "monorepo/bin-tag-manager/models/tag"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const s1TagID = "0b3c1a52-0001-11f0-8a7f-3b1d9c1f0a01"

func Test_PostTags_Errors(t *testing.T) {
	tests := []s1Case{
		s1Unauth("POST", "/tags", `{"name":"n","detail":"d"}`),
		s1BadJSON("POST", "/tags"),
		s1PermDenied("POST", "/tags", `{"name":"n","detail":"d"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().TagCreate(gomock.Any(), a, "n", "d").Return(nil, err)
		}),
		{
			name:   "success returns the tag json",
			method: "POST",
			path:   "/tags",
			body:   `{"name":"n","detail":"d"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TagCreate(gomock.Any(), a, "n", "d").Return(&tmtag.WebhookMessage{
					Identity: commonidentity.Identity{ID: s1ID(s1TagID)},
					Name:     "n",
					Detail:   "d",
				}, nil)
			},
			wantBody: `{"id":"0b3c1a52-0001-11f0-8a7f-3b1d9c1f0a01","customer_id":"00000000-0000-0000-0000-000000000000","name":"n","detail":"d","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}
	s1Run(t, tests)
}

func Test_DeleteTagsId_Errors(t *testing.T) {
	path := "/tags/" + s1TagID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TagDelete(gomock.Any(), a, s1ID(s1TagID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/tags/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", expect),
		s1NotFound("DELETE", path, "", expect),
	})
}

func Test_GetTagsId_Errors(t *testing.T) {
	path := "/tags/" + s1TagID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().TagGet(gomock.Any(), a, s1ID(s1TagID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/tags/not-a-uuid", ""),
		s1PermDenied("GET", path, "", expect),
		s1NotFound("GET", path, "", expect),
	})
}

func Test_PutTagsId_Errors(t *testing.T) {
	path := "/tags/" + s1TagID
	body := `{"name":"n2","detail":"d2"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadID("PUT", "/tags/not-a-uuid", body),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().TagUpdate(gomock.Any(), a, s1ID(s1TagID), "n2", "d2").Return(nil, err)
		}),
	})
}

func Test_GetTags_Errors(t *testing.T) {
	tests := []s1Case{
		s1Unauth("GET", "/tags", ""),
		{
			name:   "service error is propagated",
			method: "GET",
			path:   "/tags?page_size=20&page_token=tok",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TagList(gomock.Any(), a, uint64(20), "tok").Return(nil, fmt.Errorf("%w: boom", serviceerrors.ErrServiceUnavailable))
			},
			wantStatus: cerrors.StatusUnavailable,
			wantReason: "SERVICE_UNAVAILABLE",
		},
		{
			name:   "page size above max is clamped to 100",
			method: "GET",
			path:   "/tags?page_size=500",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().TagList(gomock.Any(), a, uint64(100), "").Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
		{
			name:   "next page token comes from the last tm_create",
			method: "GET",
			path:   "/tags",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				created := s1Time("2020-09-20T03:23:20.995000Z")
				m.EXPECT().TagList(gomock.Any(), a, uint64(100), "").Return([]*tmtag.WebhookMessage{
					{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(s1TagID)}, TMCreate: created},
				}, nil)
			},
			wantBody: `{"result":[{"id":"0b3c1a52-0001-11f0-8a7f-3b1d9c1f0a01","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":"2020-09-20T03:23:20.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:20.995000Z"}`,
		},
	}
	s1Run(t, tests)
}
