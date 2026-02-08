package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentContactCreate(t *testing.T) {

	type test struct {
		name string

		agent        *amagent.Agent
		firstName    string
		lastName     string
		displayName  string
		company      string
		jobTitle     string
		source       string
		externalID   string
		notes        string
		phoneNumbers []cmrequest.PhoneNumberCreate
		emails       []cmrequest.EmailCreate
		tagIDs       []uuid.UUID

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			firstName:    "John",
			lastName:     "Doe",
			displayName:  "John Doe",
			company:      "Acme",
			jobTitle:     "Engineer",
			source:       "api",
			externalID:   "ext-123",
			notes:        "test note",
			phoneNumbers: []cmrequest.PhoneNumberCreate{},
			emails:       []cmrequest.EmailCreate{},
			tagIDs:       []uuid.UUID{},

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Company:     "Acme",
				JobTitle:    "Engineer",
				Source:      "api",
				ExternalID:  "ext-123",
				Notes:       "test note",
				TMCreate:    timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName:   "John",
				LastName:    "Doe",
				DisplayName: "John Doe",
				Company:     "Acme",
				JobTitle:    "Engineer",
				Source:      "api",
				ExternalID:  "ext-123",
				TMCreate:    timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactCreate(
				ctx,
				tt.agent.CustomerID,
				tt.firstName,
				tt.lastName,
				tt.displayName,
				tt.company,
				tt.jobTitle,
				tt.source,
				tt.externalID,
				tt.notes,
				tt.phoneNumbers,
				tt.emails,
				tt.tagIDs,
			).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactCreate(
				ctx,
				tt.agent,
				tt.firstName,
				tt.lastName,
				tt.displayName,
				tt.company,
				tt.jobTitle,
				tt.source,
				tt.externalID,
				tt.notes,
				tt.phoneNumbers,
				tt.emails,
				tt.tagIDs,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactGet(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactGet(ctx, tt.agent, tt.contactID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactList(t *testing.T) {

	type test struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		responseContacts []cmcontact.Contact
		expectRes        []*cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			size:  10,
			token: "2021-03-01T01:00:00.995000Z",
			filters: map[string]string{
				"customer_id": "5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9",
				"deleted":     "false",
			},

			responseContacts: []cmcontact.Contact{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
						CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
					},
					FirstName: "John",
					LastName:  "Doe",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
						CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
					},
					FirstName: "Jane",
					LastName:  "Smith",
				},
			},
			expectRes: []*cmcontact.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
						CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
					},
					FirstName: "John",
					LastName:  "Doe",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
						CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
					},
					FirstName: "Jane",
					LastName:  "Smith",
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

			expectFilters := map[cmcontact.Field]any{
				cmcontact.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				cmcontact.FieldDeleted:    false,
			}
			mockReq.EXPECT().ContactV1ContactList(ctx, tt.token, tt.size, expectFilters).Return(tt.responseContacts, nil)

			res, err := h.ServiceAgentContactList(ctx, tt.agent, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactUpdate(t *testing.T) {

	type test struct {
		name string

		agent       *amagent.Agent
		contactID   uuid.UUID
		firstName   *string
		lastName    *string
		displayName *string
		company     *string
		jobTitle    *string
		externalID  *string
		notes       *string

		responseContactGet    *cmcontact.Contact
		responseContactUpdate *cmcontact.Contact
		expectRes             *cmcontact.WebhookMessage
	}

	firstName := "Updated"
	lastName := "Name"

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			firstName: &firstName,
			lastName:  &lastName,

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContactUpdate: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "Updated",
				LastName:  "Name",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "Updated",
				LastName:  "Name",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1ContactUpdate(
				ctx,
				tt.contactID,
				tt.firstName,
				tt.lastName,
				tt.displayName,
				tt.company,
				tt.jobTitle,
				tt.externalID,
				tt.notes,
			).Return(tt.responseContactUpdate, nil)

			res, err := h.ServiceAgentContactUpdate(ctx, tt.agent, tt.contactID, tt.firstName, tt.lastName, tt.displayName, tt.company, tt.jobTitle, tt.externalID, tt.notes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactDelete(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID

		responseContactGet    *cmcontact.Contact
		responseContactDelete *cmcontact.Contact
		expectRes             *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContactDelete: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				TMDelete: timePtr("2020-09-20T04:00:00.000000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				TMDelete: timePtr("2020-09-20T04:00:00.000000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1ContactDelete(ctx, tt.contactID).Return(tt.responseContactDelete, nil)

			res, err := h.ServiceAgentContactDelete(ctx, tt.agent, tt.contactID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactLookup(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		phoneE164 string
		email     string

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "lookup by phone",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			phoneE164: "+15551234567",
			email:     "",

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
		},
		{
			name: "lookup by email",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			phoneE164: "",
			email:     "john@example.com",

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactLookup(ctx, tt.agent.CustomerID, tt.phoneE164, tt.email).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactLookup(ctx, tt.agent, tt.phoneE164, tt.email)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactPhoneNumberCreate(t *testing.T) {

	type test struct {
		name string

		agent      *amagent.Agent
		contactID  uuid.UUID
		number     string
		numberE164 string
		phoneType  string
		isPrimary  bool

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID:  uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			number:     "+15551234567",
			numberE164: "",
			phoneType:  "mobile",
			isPrimary:  true,

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:        uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
						Number:    "+15551234567",
						Type:      "mobile",
						IsPrimary: true,
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:        uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
						Number:    "+15551234567",
						Type:      "mobile",
						IsPrimary: true,
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1PhoneNumberCreate(ctx, tt.contactID, tt.number, tt.numberE164, tt.phoneType, tt.isPrimary).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactPhoneNumberCreate(ctx, tt.agent, tt.contactID, tt.number, tt.numberE164, tt.phoneType, tt.isPrimary)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactPhoneNumberUpdate(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		contactID     uuid.UUID
		phoneNumberID uuid.UUID
		fields        map[string]any

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID:     uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			phoneNumberID: uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
			fields: map[string]any{
				"number": "+15559999999",
			},

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
						Number: "+15559999999",
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				PhoneNumbers: []cmcontact.PhoneNumber{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
						Number: "+15559999999",
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1PhoneNumberUpdate(ctx, tt.contactID, tt.phoneNumberID, tt.fields).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactPhoneNumberUpdate(ctx, tt.agent, tt.contactID, tt.phoneNumberID, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactPhoneNumberDelete(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		contactID     uuid.UUID
		phoneNumberID uuid.UUID

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID:     uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			phoneNumberID: uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1PhoneNumberDelete(ctx, tt.contactID, tt.phoneNumberID).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactPhoneNumberDelete(ctx, tt.agent, tt.contactID, tt.phoneNumberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactEmailCreate(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID
		address   string
		emailType string
		isPrimary bool

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			address:   "john@example.com",
			emailType: "work",
			isPrimary: true,

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Emails: []cmcontact.Email{
					{
						ID:        uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
						Address:   "john@example.com",
						Type:      "work",
						IsPrimary: true,
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Emails: []cmcontact.Email{
					{
						ID:        uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
						Address:   "john@example.com",
						Type:      "work",
						IsPrimary: true,
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1EmailCreate(ctx, tt.contactID, tt.address, tt.emailType, tt.isPrimary).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactEmailCreate(ctx, tt.agent, tt.contactID, tt.address, tt.emailType, tt.isPrimary)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactEmailUpdate(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID
		emailID   uuid.UUID
		fields    map[string]any

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			emailID:   uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
			fields: map[string]any{
				"address": "updated@example.com",
			},

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Emails: []cmcontact.Email{
					{
						ID:      uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
						Address: "updated@example.com",
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Emails: []cmcontact.Email{
					{
						ID:      uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),
						Address: "updated@example.com",
					},
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1EmailUpdate(ctx, tt.contactID, tt.emailID, tt.fields).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactEmailUpdate(ctx, tt.agent, tt.contactID, tt.emailID, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactEmailDelete(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID
		emailID   uuid.UUID

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			emailID:   uuid.FromStringOrNil("b2c3d4e5-0002-11ec-0002-000000000002"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1EmailDelete(ctx, tt.contactID, tt.emailID).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactEmailDelete(ctx, tt.agent, tt.contactID, tt.emailID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactTagAdd(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID
		tagID     uuid.UUID

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			tagID:     uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1TagAdd(ctx, tt.contactID, tt.tagID).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactTagAdd(ctx, tt.agent, tt.contactID, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentContactTagRemove(t *testing.T) {

	type test struct {
		name string

		agent     *amagent.Agent
		contactID uuid.UUID
		tagID     uuid.UUID

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			contactID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
			tagID:     uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1TagRemove(ctx, tt.contactID, tt.tagID).Return(tt.responseContact, nil)

			res, err := h.ServiceAgentContactTagRemove(ctx, tt.agent, tt.contactID, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
