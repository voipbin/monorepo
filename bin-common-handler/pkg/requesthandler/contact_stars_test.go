package requesthandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcontactstar "monorepo/bin-contact-manager/models/contactstar"
)

func Test_ContactV1ContactStar(t *testing.T) {
	customerID := uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3")
	agentID := uuid.FromStringOrNil("11111111-2c74-11ee-98fb-0762519529f3")
	contactID := uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f")
	created := time.Date(2026, 6, 1, 9, 0, 0, 123456000, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}
	ctx := context.Background()

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", &sock.Request{
		URI:      "/v1/contacts/5623e25e-2c74-11ee-87a6-bfa8ae34077f/stars/11111111-2c74-11ee-98fb-0762519529f3",
		Method:   sock.RequestMethodPost,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
	}).Return(&sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e","customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","agent_id":"11111111-2c74-11ee-98fb-0762519529f3","contact_id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f","tm_create":"2026-06-01T09:00:00.123456Z"}`),
	}, nil)

	res, err := reqHandler.ContactV1ContactStar(ctx, customerID, agentID, contactID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := &cmcontactstar.ContactStar{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"), CustomerID: customerID},
		AgentID:   agentID,
		ContactID: contactID,
		TMCreate:  &created,
	}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_ContactV1ContactStar_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", gomock.Any()).Return(&sock.Response{StatusCode: 404}, nil)

	if res, err := reqHandler.ContactV1ContactStar(context.Background(), uuid.Nil, uuid.Nil, uuid.Nil); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
	}
}

func Test_ContactV1ContactUnstar(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", &sock.Request{
		URI:      "/v1/contacts/5623e25e-2c74-11ee-87a6-bfa8ae34077f/stars/11111111-2c74-11ee-98fb-0762519529f3",
		Method:   sock.RequestMethodDelete,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3"}`),
	}).Return(&sock.Response{StatusCode: 200}, nil)

	if err := reqHandler.ContactV1ContactUnstar(context.Background(),
		uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
		uuid.FromStringOrNil("11111111-2c74-11ee-98fb-0762519529f3"),
		uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f")); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", gomock.Any()).Return(&sock.Response{StatusCode: 404}, nil)
	if err := reqHandler.ContactV1ContactUnstar(context.Background(), uuid.Nil, uuid.Nil, uuid.Nil); err == nil {
		t.Error("Wrong match. expect: error")
	}
}

func Test_ContactV1ContactStarredList(t *testing.T) {
	tests := []struct {
		name string

		size  uint64
		token string

		expectURI       string
		response        *sock.Response
		expectIDs       []uuid.UUID
		expectNextToken string
	}{
		{
			name:      "normal with paging",
			size:      10,
			token:     "2026-06-01T09:30:01.000000Z",
			expectURI: "/v1/contacts/starred?page_size=10&page_token=2026-06-01T09%3A30%3A01.000000Z",
			response: &sock.Response{StatusCode: 200, DataType: "application/json",
				Data: []byte(`[{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e","tm_create":"2026-06-01T09:00:00.123456Z"},{"id":"9fe6c136-2c75-11ee-a3a4-37400837e12e","tm_create":"2026-06-01T08:00:00.000001Z"}]`)},
			expectIDs: []uuid.UUID{
				uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
				uuid.FromStringOrNil("9fe6c136-2c75-11ee-a3a4-37400837e12e"),
			},
			// the token is the CONTACT's tm_create of the last item
			expectNextToken: "2026-06-01T08:00:00.000001Z",
		},
		{
			name:            "empty result has no token and no query",
			expectURI:       "/v1/contacts/starred",
			response:        &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`[]`)},
			expectNextToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", &sock.Request{
				URI:      tt.expectURI,
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","agent_id":"11111111-2c74-11ee-98fb-0762519529f3"}`),
			}).Return(tt.response, nil)

			res, next, err := reqHandler.ContactV1ContactStarredList(context.Background(),
				uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
				uuid.FromStringOrNil("11111111-2c74-11ee-98fb-0762519529f3"),
				tt.size, tt.token)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var ids []uuid.UUID
			for _, c := range res {
				ids = append(ids, c.ID)
			}
			if !reflect.DeepEqual(ids, tt.expectIDs) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectIDs, ids)
			}
			if next != tt.expectNextToken {
				t.Errorf("Wrong next token. expect: %q, got: %q", tt.expectNextToken, next)
			}
		})
	}
}

func Test_ContactV1ContactStarList(t *testing.T) {
	tests := []struct {
		name string

		size  uint64
		token string

		expectURI       string
		response        *sock.Response
		expectContacts  []uuid.UUID
		expectNextToken string
	}{
		{
			name:      "normal with paging",
			size:      3,
			token:     "2026-06-02T10:00:00.000000Z",
			expectURI: "/v1/contact_stars?page_size=3&page_token=2026-06-02T10%3A00%3A00.000000Z",
			response: &sock.Response{StatusCode: 200, DataType: "application/json",
				Data: []byte(`[{"contact_id":"5623e25e-2c74-11ee-87a6-bfa8ae34077f","tm_create":"2026-06-01T09:00:00.123456Z"}]`)},
			expectContacts:  []uuid.UUID{uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f")},
			expectNextToken: "2026-06-01T09:00:00.123456Z",
		},
		{
			name:            "empty",
			expectURI:       "/v1/contact_stars",
			response:        &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`[]`)},
			expectNextToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{sock: mockSock}

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", &sock.Request{
				URI:      tt.expectURI,
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","agent_id":"11111111-2c74-11ee-98fb-0762519529f3"}`),
			}).Return(tt.response, nil)

			res, next, err := reqHandler.ContactV1ContactStarList(context.Background(),
				uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
				uuid.FromStringOrNil("11111111-2c74-11ee-98fb-0762519529f3"),
				tt.size, tt.token)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var ids []uuid.UUID
			for _, s := range res {
				ids = append(ids, s.ContactID)
			}
			if !reflect.DeepEqual(ids, tt.expectContacts) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectContacts, ids)
			}
			if next != tt.expectNextToken {
				t.Errorf("Wrong next token. expect: %q, got: %q", tt.expectNextToken, next)
			}
		})
	}
}
