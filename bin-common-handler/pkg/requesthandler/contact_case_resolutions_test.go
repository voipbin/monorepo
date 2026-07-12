package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmresolution "monorepo/bin-contact-manager/models/resolution"
)

func Test_ContactV1CaseResolutionCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		caseID         uuid.UUID
		contactID      uuid.UUID
		resolutionType string
		resolvedByType string
		resolvedByID   uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cmresolution.Resolution
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f3"),
			caseID:         uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			contactID:      uuid.FromStringOrNil("6623e25e-2c74-11ee-87a6-bfa8ae34077f"),
			resolutionType: "positive",
			resolvedByType: "agent",
			resolvedByID:   uuid.FromStringOrNil("7623e25e-2c74-11ee-87a6-bfa8ae34077f"),

			expectTarget: "bin-manager.contact-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae34077f/resolutions",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data: []byte(
					`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f3","contact_id":"6623e25e-2c74-11ee-87a6-bfa8ae34077f","resolution_type":"positive","resolved_by_type":"agent","resolved_by_id":"7623e25e-2c74-11ee-87a6-bfa8ae34077f"}`,
				),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8fe6c136-2c75-11ee-a3a4-37400837e12e"}`),
			},
			expectRes: &cmresolution.Resolution{
				ID: uuid.FromStringOrNil("8fe6c136-2c75-11ee-a3a4-37400837e12e"),
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

			res, err := reqHandler.ContactV1CaseResolutionCreate(ctx, tt.customerID, tt.caseID, tt.contactID, tt.resolutionType, tt.resolvedByType, tt.resolvedByID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactV1CaseResolutionCreate_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f4")
	caseID := uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae340780")
	contactID := uuid.FromStringOrNil("6623e25e-2c74-11ee-87a6-bfa8ae340781")
	agentID := uuid.FromStringOrNil("7623e25e-2c74-11ee-87a6-bfa8ae340782")

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", gomock.Any()).Return(&sock.Response{StatusCode: 404}, nil)

	res, err := reqHandler.ContactV1CaseResolutionCreate(ctx, customerID, caseID, contactID, "positive", "agent", agentID)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: %v", res)
	}
}

func Test_ContactV1CaseResolutionDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{sock: mockSock}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("55ecfc4e-2c74-11ee-98fb-0762519529f5")
	caseID := uuid.FromStringOrNil("5623e25e-2c74-11ee-87a6-bfa8ae340783")
	resolutionID := uuid.FromStringOrNil("6623e25e-2c74-11ee-87a6-bfa8ae340784")

	expectRequest := &sock.Request{
		URI:      "/v1/cases/5623e25e-2c74-11ee-87a6-bfa8ae340783/resolutions/6623e25e-2c74-11ee-87a6-bfa8ae340784",
		Method:   sock.RequestMethodDelete,
		DataType: ContentTypeJSON,
		Data:     []byte(`{"customer_id":"55ecfc4e-2c74-11ee-98fb-0762519529f5"}`),
	}
	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.contact-manager.request", expectRequest).Return(&sock.Response{StatusCode: 200}, nil)

	if err := reqHandler.ContactV1CaseResolutionDelete(ctx, customerID, caseID, resolutionID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
