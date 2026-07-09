package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
)

func Test_ContactV1CaseList(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		status     string
		ownerType  string
		ownerID    uuid.UUID
		contactID  uuid.UUID
		size       uint64
		token      string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []*cmkase.Case
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			status:     "open",
			ownerType:  "agent",
			ownerID:    uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases?owner_id=5623e25e-2c74-11ee-87a6-bfa8ae34077f&owner_type=agent&status=open",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"}]`),
			},
			expectRes: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
				},
			},
		},
		{
			name: "contact_id filter",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			contactID:  uuid.FromStringOrNil("7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases?contact_id=7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"}]`),
			},
			expectRes: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
				},
			},
		},
		{
			name: "page_size and page_token are encoded into the query string",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			size:       uint64(50),
			token:      "2026-06-28T10:00:00.000000Z",

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases?page_size=50&page_token=2026-06-28T10%3A00%3A00.000000Z",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"}]`),
			},
			expectRes: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, nextToken, err := reqHandler.ContactV1CaseList(ctx, tt.customerID, tt.status, tt.ownerType, tt.ownerID, tt.contactID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
			if nextToken != "" {
				t.Errorf("Wrong match. expect: empty, got: %v", nextToken)
			}
		})
	}
}

// Test_ContactV1CaseList_NextToken verifies nextToken is derived from
// the last returned item's tm_create (round-2-review-fixed pagination
// wiring), matching ContactV1InteractionList's convention.
func Test_ContactV1CaseList_NextToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	customerID := uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3")

	ctx := context.Background()
	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", &sock.Request{
		URI:      "/v1/cases?",
		Method:   sock.RequestMethodGet,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
	}).Return(&sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`[{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f","tm_create":"2026-06-28T12:00:00.000000Z"},{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e","tm_create":"2026-06-28T11:00:00.000000Z"}]`),
	}, nil)

	_, nextToken, err := reqHandler.ContactV1CaseList(ctx, customerID, "", "", uuid.Nil, uuid.Nil, 0, "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if nextToken != "2026-06-28T11:00:00.000000Z" {
		t.Errorf("nextToken = %q, want %q (last item's tm_create)", nextToken, "2026-06-28T11:00:00.000000Z")
	}
}

func Test_ContactV1CaseListUnresolved(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []*cmkase.Case
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/unresolved",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"}]`),
			},
			expectRes: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, nextToken, err := reqHandler.ContactV1CaseListUnresolved(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
			if nextToken != "" {
				t.Errorf("Wrong match. expect: empty, got: %v", nextToken)
			}
		})
	}
}

func Test_ContactV1CaseGet(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		id         uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cmkase.Case
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			id:         uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"}`),
			},
			expectRes: &cmkase.Case{
				ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ContactV1CaseGet(ctx, tt.customerID, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseClose(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		id           uuid.UUID
		closedByType string
		closedByID   uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cmkase.Case
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			id:           uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			closedByType: "agent",
			closedByID:   uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/close",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","closed_by_type":"agent","closed_by_id":"8fe6c136-2c75-11ee-a3a4-37400837e12e"}`,
				),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"Case":{"id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f"},"ClosedReason":"agent_closed","ClosedByType":"agent","AlreadyClosed":false}`),
			},
			expectRes: &cmkase.Case{
				ID: uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ContactV1CaseClose(ctx, tt.customerID, tt.id, tt.closedByType, tt.closedByID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseContinue(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		id            uuid.UUID
		callerType    string
		callerID      uuid.UUID
		callerIsAdmin bool

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cmkase.Case
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			id:            uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			callerType:    "agent",
			callerID:      uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
			callerIsAdmin: false,

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/continue",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","caller_type":"agent","caller_id":"8fe6c136-2c75-11ee-a3a4-37400837e12e","caller_is_admin":false}`,
				),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"}`),
			},
			expectRes: &cmkase.Case{
				ID: uuid.FromStringOrNil("8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ContactV1CaseContinue(ctx, tt.customerID, tt.id, tt.callerType, tt.callerID, tt.callerIsAdmin)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseNoteList(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []*cmcasenote.CaseNote
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/notes",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e"}]`),
			},
			expectRes: []*cmcasenote.CaseNote{
				{
					ID: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ContactV1CaseNoteList(ctx, tt.customerID, tt.caseID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseNoteCreate(t *testing.T) {

	authorID := uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e")

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID
		authorType string
		authorID   *uuid.UUID
		text       string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cmcasenote.CaseNote
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			authorType: "agent",
			authorID:   &authorID,
			text:       "test note",

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/notes",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","author_type":"agent","author_id":"8fe6c136-2c75-11ee-a3a4-37400837e12e","text":"test note"}`,
				),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"}`),
			},
			expectRes: &cmcasenote.CaseNote{
				ID: uuid.FromStringOrNil("8fb8f1fc-2c75-11ee-aa2c-cb07baf2171a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ContactV1CaseNoteCreate(ctx, tt.customerID, tt.caseID, tt.authorType, tt.authorID, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseNoteDelete(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID
		noteID     uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			noteID:     uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/notes/8fe6c136-2c75-11ee-a3a4-37400837e12e",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.ContactV1CaseNoteDelete(ctx, tt.customerID, tt.caseID, tt.noteID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ContactV1CaseTagList(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []uuid.UUID
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/tags",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`["8fe6c136-2c75-11ee-a3a4-37400837e12e"]`),
			},
			expectRes: []uuid.UUID{
				uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ContactV1CaseTagList(ctx, tt.customerID, tt.caseID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseTagAdd(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID
		tagID      uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			tagID:      uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/tags",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","tag_id":"8fe6c136-2c75-11ee-a3a4-37400837e12e"}`,
				),
			},
			response: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.ContactV1CaseTagAdd(ctx, tt.customerID, tt.caseID, tt.tagID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ContactV1CaseTagRemove(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID
		tagID      uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			tagID:      uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/tags/8fe6c136-2c75-11ee-a3a4-37400837e12e",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.ContactV1CaseTagRemove(ctx, tt.customerID, tt.caseID, tt.tagID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
