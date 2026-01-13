package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	ememail "monorepo/bin-email-manager/models/email"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_EmailSend(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		destinations []commonaddress.Address
		subject      string
		content      string
		attachments  []ememail.Attachment

		response  *ememail.Email
		expectRes *ememail.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			destinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			subject: "test subject",
			content: "test content",
			attachments: []ememail.Attachment{
				{
					ReferenceType: ememail.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("04fcf8b0-00ea-11f0-b8f0-afcdc7672657"),
				},
			},

			response: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("05945732-00ea-11f0-b9b1-eb344fffb9ac"),
				},
			},
			expectRes: &ememail.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("05945732-00ea-11f0-b9b1-eb344fffb9ac"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().EmailV1EmailSend(ctx, tt.agent.CustomerID, uuid.Nil, tt.destinations, tt.subject, tt.content, tt.attachments.Return(tt.response, nil)
			res, err := h.EmailSend(ctx, tt.agent, tt.destinations, tt.subject, tt.content, tt.attachments)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailGets(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseEmails []ememail.Email
		expectFilters  map[string]string
		expectRes      []*ememail.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			pageToken: "2020-10-20T01:00:00.995000",
			pageSize:  10,

			responseEmails: []ememail.Email{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("05ed5110-00eb-11f0-aa87-77268fca0f1e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("06121f22-00eb-11f0-8358-d7f6cbb3a076"),
					},
				},
			},
			expectFilters: map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			expectRes: []*ememail.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("05ed5110-00eb-11f0-aa87-77268fca0f1e"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("06121f22-00eb-11f0-8358-d7f6cbb3a076"),
					},
				},
			},
		},
		{
			name: "1 action",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			pageToken: "2020-10-20T01:00:00.995000",
			pageSize:  10,

			responseEmails: []ememail.Email{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("06370d0a-00eb-11f0-8c7f-f77746cdfc43"),
					},
				},
			},
			expectFilters: map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			expectRes: []*ememail.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("06370d0a-00eb-11f0-8c7f-f77746cdfc43"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().EmailV1EmailGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters.Return(tt.responseEmails, nil)

			res, err := h.EmailGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailGet(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		emailID uuid.UUID

		response  *ememail.Email
		expectRes *ememail.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			emailID: uuid.FromStringOrNil("917887a4-00eb-11f0-9422-7fe52e96709b"),

			response: &ememail.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("917887a4-00eb-11f0-9422-7fe52e96709b"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
			expectRes: &ememail.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("917887a4-00eb-11f0-9422-7fe52e96709b"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().EmailV1EmailGet(ctx, tt.emailID.Return(tt.response, nil)

			res, err := h.EmailGet(ctx, tt.agent, tt.emailID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		emailID uuid.UUID

		responseEmail *ememail.Email
		expectRes     *ememail.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			emailID: uuid.FromStringOrNil("150a839c-00ec-11f0-87c5-536f3d24ce8b"),

			responseEmail: &ememail.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("150a839c-00ec-11f0-87c5-536f3d24ce8b"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
			expectRes: &ememail.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("150a839c-00ec-11f0-87c5-536f3d24ce8b"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().EmailV1EmailGet(ctx, tt.emailID.Return(tt.responseEmail, nil)
			mockReq.EXPECT().EmailV1EmailDelete(ctx, tt.emailID.Return(tt.responseEmail, nil)

			res, err := h.EmailDelete(ctx, tt.agent, tt.emailID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
