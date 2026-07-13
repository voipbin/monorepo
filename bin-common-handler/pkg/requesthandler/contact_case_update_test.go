package requesthandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmkase "monorepo/bin-contact-manager/models/kase"
)

func Test_ContactV1CaseUpdateContact(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		caseID     uuid.UUID
		contactID  uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cmkase.Case
	}{
		{
			name: "attach case to contact",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			contactID:  uuid.FromStringOrNil("7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","contact_id":"7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111"}`,
				),
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
		{
			name: "detach case from contact",

			customerID: uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:     uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			contactID:  uuid.Nil,

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","contact_id":"00000000-0000-0000-0000-000000000000"}`,
				),
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

			res, err := reqHandler.ContactV1CaseUpdateContact(ctx, tt.customerID, tt.caseID, tt.contactID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseUpdateContact_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	customerID := uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3")
	caseID := uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f")
	contactID := uuid.FromStringOrNil("7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111")

	ctx := context.Background()
	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", &sock.Request{
		URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f",
		Method:   sock.RequestMethodPut,
		DataType: ContentTypeJSON,
		Data: []byte(
			`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","contact_id":"7a3ec8f0-2c74-11ee-b0e5-8f2ac8c9a111"}`,
		),
	}).Return(nil, fmt.Errorf("connection refused"))

	res, err := reqHandler.ContactV1CaseUpdateContact(ctx, customerID, caseID, contactID)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}
