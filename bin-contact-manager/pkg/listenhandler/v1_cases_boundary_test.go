package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_ProcessV1Cases_BoundaryAndErrorMapping(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-0061-0061-0061-000000000001")
	caseID := uuid.FromStringOrNil("22222222-0061-0061-0061-000000000002")
	subID := uuid.FromStringOrNil("22222222-0061-0061-0061-000000000003")
	ownerID := uuid.FromStringOrNil("22222222-0061-0061-0061-000000000004")

	type fn func(h *listenHandler, ctx context.Context, req *sock.Request) (*sock.Response, error)

	tests := []struct {
		name   string
		call   fn
		uri    string
		method sock.RequestMethod
		body   string
		// bodyRequired is false for GET style handlers that ignore a malformed body.
		bodyRequired bool
		// subRequired is true when the URI ends with a sub resource id.
		subRequired bool

		expect func(m *casehandler.MockCaseHandler, err error)
	}{
		{
			name:         "close",
			call:         (*listenHandler).processV1CasesIDClosePost,
			uri:          "/v1/cases/" + caseID.String() + "/close",
			method:       sock.RequestMethodPost,
			body:         fmt.Sprintf(`{"customer_id":"%s","closed_by_type":"agent","closed_by_id":"%s"}`, customerID, ownerID),
			bodyRequired: true,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().Close(gomock.Any(), customerID, caseID, commonidentity.OwnerTypeAgent, ownerID).Return(nil, err)
			},
		},
		{
			name:         "assign",
			call:         (*listenHandler).processV1CasesIDAssignPost,
			uri:          "/v1/cases/" + caseID.String() + "/assign",
			method:       sock.RequestMethodPost,
			body:         fmt.Sprintf(`{"customer_id":"%s","owner_type":"agent","owner_id":"%s"}`, customerID, ownerID),
			bodyRequired: true,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().Assign(gomock.Any(), customerID, caseID, commonidentity.OwnerTypeAgent, ownerID).Return(nil, err)
			},
		},
		{
			name:         "notes get",
			call:         (*listenHandler).processV1CasesIDNotesGet,
			uri:          "/v1/cases/" + caseID.String() + "/notes",
			method:       sock.RequestMethodGet,
			body:         fmt.Sprintf(`{"customer_id":"%s"}`, customerID),
			bodyRequired: false,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().CaseNoteListByCase(gomock.Any(), customerID, caseID).Return(nil, err)
			},
		},
		{
			name:         "notes post",
			call:         (*listenHandler).processV1CasesIDNotesPost,
			uri:          "/v1/cases/" + caseID.String() + "/notes",
			method:       sock.RequestMethodPost,
			body:         fmt.Sprintf(`{"customer_id":"%s","author_type":"agent","text":"note text"}`, customerID),
			bodyRequired: true,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().CaseNoteCreate(gomock.Any(), customerID, caseID, "agent", (*uuid.UUID)(nil), "note text").Return(nil, err)
			},
		},
		{
			name:         "notes id delete",
			call:         (*listenHandler).processV1CasesIDNotesIDDelete,
			uri:          "/v1/cases/" + caseID.String() + "/notes/" + subID.String(),
			method:       sock.RequestMethodDelete,
			body:         fmt.Sprintf(`{"customer_id":"%s"}`, customerID),
			bodyRequired: true,
			subRequired:  true,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().CaseNoteDelete(gomock.Any(), customerID, caseID, subID).Return(err)
			},
		},
		{
			name:         "tags get",
			call:         (*listenHandler).processV1CasesIDTagsGet,
			uri:          "/v1/cases/" + caseID.String() + "/tags",
			method:       sock.RequestMethodGet,
			body:         fmt.Sprintf(`{"customer_id":"%s"}`, customerID),
			bodyRequired: false,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().CaseTagList(gomock.Any(), customerID, caseID).Return(nil, err)
			},
		},
		{
			name:         "tags post",
			call:         (*listenHandler).processV1CasesIDTagsPost,
			uri:          "/v1/cases/" + caseID.String() + "/tags",
			method:       sock.RequestMethodPost,
			body:         fmt.Sprintf(`{"customer_id":"%s","tag_id":"%s"}`, customerID, subID),
			bodyRequired: true,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().CaseTagAdd(gomock.Any(), customerID, caseID, subID).Return(err)
			},
		},
		{
			name:         "tags id delete",
			call:         (*listenHandler).processV1CasesIDTagsIDDelete,
			uri:          "/v1/cases/" + caseID.String() + "/tags/" + subID.String(),
			method:       sock.RequestMethodDelete,
			body:         fmt.Sprintf(`{"customer_id":"%s"}`, customerID),
			bodyRequired: true,
			subRequired:  true,
			expect: func(m *casehandler.MockCaseHandler, err error) {
				m.EXPECT().CaseTagRemove(gomock.Any(), customerID, caseID, subID).Return(err)
			},
		},
	}

	for _, tt := range tests {
		// rejected input must never reach the case handler (strict gomock, no expectation).
		reject := func(t *testing.T, uri string, body string) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _ := newTestListenHandlerWithCase(mc)

			res, err := tt.call(h, context.Background(), &sock.Request{URI: uri, Method: tt.method, Data: []byte(body)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("StatusCode = %d, want 400", res.StatusCode)
			}
		}

		t.Run(tt.name+" invalid case id", func(t *testing.T) {
			reject(t, strings.Replace(tt.uri, caseID.String(), "not-a-uuid", 1), tt.body)
		})
		if tt.subRequired {
			t.Run(tt.name+" invalid sub id", func(t *testing.T) {
				reject(t, strings.Replace(tt.uri, subID.String(), "not-a-uuid", 1), tt.body)
			})
		}
		t.Run(tt.name+" missing customer_id", func(t *testing.T) {
			reject(t, tt.uri, `{}`)
		})
		if tt.bodyRequired {
			t.Run(tt.name+" malformed body", func(t *testing.T) {
				reject(t, tt.uri, `{bad`)
			})
		} else {
			t.Run(tt.name+" malformed body is treated as missing customer_id", func(t *testing.T) {
				reject(t, tt.uri, `{bad`)
			})
		}

		errCases := []struct {
			name   string
			err    error
			expect int
		}{
			{"not found", dbhandler.ErrNotFound, 404},
			{"unclassified", fmt.Errorf("storage failure"), 500},
		}
		for _, ec := range errCases {
			t.Run(tt.name+" downstream "+ec.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, mockCase := newTestListenHandlerWithCase(mc)
				tt.expect(mockCase, ec.err)

				res, err := tt.call(h, context.Background(), &sock.Request{URI: tt.uri, Method: tt.method, Data: []byte(tt.body)})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.StatusCode != ec.expect {
					t.Errorf("StatusCode = %d, want %d", res.StatusCode, ec.expect)
				}
			})
		}
	}
}

func Test_ProcessV1Cases_SuccessBodies(t *testing.T) {
	customerID := uuid.FromStringOrNil("22222222-0062-0062-0062-000000000001")
	caseID := uuid.FromStringOrNil("22222222-0062-0062-0062-000000000002")
	ownerID := uuid.FromStringOrNil("22222222-0062-0062-0062-000000000003")
	noteID := uuid.FromStringOrNil("22222222-0062-0062-0062-000000000004")
	tagID := uuid.FromStringOrNil("22222222-0062-0062-0062-000000000005")

	t.Run("assign returns the updated case", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)

		mockCase.EXPECT().Assign(gomock.Any(), customerID, caseID, commonidentity.OwnerTypeAgent, ownerID).Return(&kase.Case{ID: caseID, CustomerID: customerID, OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID}, nil)

		body := fmt.Sprintf(`{"customer_id":"%s","owner_type":"agent","owner_id":"%s"}`, customerID, ownerID)
		res, err := h.processV1CasesIDAssignPost(context.Background(), &sock.Request{URI: "/v1/cases/" + caseID.String() + "/assign", Method: sock.RequestMethodPost, Data: []byte(body)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 200 || res.DataType != "application/json" {
			t.Fatalf("response = %d %q, want 200 application/json", res.StatusCode, res.DataType)
		}
		var got kase.Case
		if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
			t.Fatalf("body is not a case: %v", errUnmarshal)
		}
		if got.ID != caseID || got.OwnerID != ownerID || got.OwnerType != commonidentity.OwnerTypeAgent {
			t.Errorf("unexpected case: %+v", got)
		}
	})

	t.Run("notes get returns the list", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)

		mockCase.EXPECT().CaseNoteListByCase(gomock.Any(), customerID, caseID).Return([]*casenote.CaseNote{{ID: noteID, CaseID: caseID, CustomerID: customerID, Text: "first"}}, nil)

		res, err := h.processV1CasesIDNotesGet(context.Background(), &sock.Request{URI: "/v1/cases/" + caseID.String() + "/notes", Method: sock.RequestMethodGet, Data: []byte(fmt.Sprintf(`{"customer_id":"%s"}`, customerID))})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got []*casenote.CaseNote
		if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil || res.StatusCode != 200 {
			t.Fatalf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
		}
		if len(got) != 1 || got[0].ID != noteID || got[0].Text != "first" {
			t.Errorf("unexpected notes: %s", res.Data)
		}
	})

	t.Run("tags get returns the ids", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockCase := newTestListenHandlerWithCase(mc)

		mockCase.EXPECT().CaseTagList(gomock.Any(), customerID, caseID).Return([]uuid.UUID{tagID}, nil)

		res, err := h.processV1CasesIDTagsGet(context.Background(), &sock.Request{URI: "/v1/cases/" + caseID.String() + "/tags", Method: sock.RequestMethodGet, Data: []byte(fmt.Sprintf(`{"customer_id":"%s"}`, customerID))})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got []uuid.UUID
		if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil || res.StatusCode != 200 {
			t.Fatalf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
		}
		if len(got) != 1 || got[0] != tagID {
			t.Errorf("unexpected tags: %s", res.Data)
		}
	})
}
