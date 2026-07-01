package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ContactCreate(t *testing.T) {

	type test struct {
		name string

		agent        *auth.AuthIdentity
		firstName    string
		lastName     string
		displayName  string
		company      string
		jobTitle     string
		source       string
		externalID   string
		notes        string
		addresses    []cmrequest.AddressCreate
		tagIDs       []uuid.UUID

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			firstName:    "John",
			lastName:     "Doe",
			displayName:  "John Doe",
			company:      "Acme",
			jobTitle:     "Engineer",
			source:       "api",
			externalID:   "ext-123",
			notes:        "test note",
			addresses: []cmrequest.AddressCreate{},
			tagIDs:       []uuid.UUID{},

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
				tt.addresses,
				tt.tagIDs,
			).Return(tt.responseContact, nil)

			res, err := h.ContactCreate(
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
				tt.addresses,
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

func Test_ContactGet(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.ContactGet(ctx, tt.agent, tt.contactID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactList(t *testing.T) {

	type test struct {
		name string

		agent   *auth.AuthIdentity
		size    uint64
		token   string
		filters map[string]string

		responseContacts []cmcontact.Contact
		expectRes        []*cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:  10,
			token: "2021-03-01T01:00:00.995000Z",
			filters: map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},

			responseContacts: []cmcontact.Contact{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					FirstName: "John",
					LastName:  "Doe",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					FirstName: "Jane",
					LastName:  "Smith",
				},
			},
			expectRes: []*cmcontact.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
					},
					FirstName: "John",
					LastName:  "Doe",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
						CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
				cmcontact.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				cmcontact.FieldDeleted:    false,
			}
			mockReq.EXPECT().ContactV1ContactList(ctx, tt.token, tt.size, expectFilters).Return(tt.responseContacts, nil)

			res, err := h.ContactList(ctx, tt.agent, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactUpdate(t *testing.T) {

	type test struct {
		name string

		agent       *auth.AuthIdentity
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

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			firstName: &firstName,
			lastName:  &lastName,

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseContactUpdate: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				FirstName: "Updated",
				LastName:  "Name",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.ContactUpdate(ctx, tt.agent, tt.contactID, tt.firstName, tt.lastName, tt.displayName, tt.company, tt.jobTitle, tt.externalID, tt.notes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactDelete(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID

		responseContactGet    *cmcontact.Contact
		responseContactDelete *cmcontact.Contact
		expectRes             *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseContactDelete: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				TMDelete: timePtr("2020-09-20T04:00:00.000000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.ContactDelete(ctx, tt.agent, tt.contactID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactLookup(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		phoneE164 string
		email     string

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			phoneE164: "+15551234567",
			email:     "",

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				FirstName: "John",
				LastName:  "Doe",
				TMCreate:  timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.ContactLookup(ctx, tt.agent, tt.phoneE164, tt.email)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactAddressCreate(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID
		addrType  string
		target    string
		isPrimary bool

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal tel",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			addrType:  "tel",
			target:    "+121****1234",
			isPrimary: false,

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Addresses: []cmcontact.Address{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
						Type:   "tel",
						Target: "+121****1234",
					},
				},
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Addresses: []cmcontact.Address{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
						Type:   "tel",
						Target: "+121****1234",
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContactGet, nil)
			mockReq.EXPECT().ContactV1AddressCreate(ctx, tt.contactID, tt.addrType, tt.target, tt.isPrimary, "", "").Return(&cmcontact.Address{}, nil)
			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)

			res, err := h.ContactAddressCreate(ctx, tt.agent, tt.contactID, tt.addrType, tt.target, tt.isPrimary, "", "")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactAddressUpdate(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID
		addressID uuid.UUID
		fields    map[string]any

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			addressID: uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
			fields:    map[string]any{"target": "+121****9999"},

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Addresses: []cmcontact.Address{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
						Type:   "tel",
						Target: "+121****9999",
					},
				},
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Addresses: []cmcontact.Address{
					{
						ID:     uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
						Type:   "tel",
						Target: "+121****9999",
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockReq.EXPECT().ContactV1AddressUpdate(ctx, tt.contactID, tt.addressID, tt.fields).Return(tt.responseContact, nil)

			res, err := h.ContactAddressUpdate(ctx, tt.agent, tt.contactID, tt.addressID, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactAddressDelete(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID
		addressID uuid.UUID

		responseContact *cmcontact.Contact
		expectRes       *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			addressID: uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),

			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			mockReq.EXPECT().ContactV1AddressDelete(ctx, tt.contactID, tt.addressID).Return(tt.responseContact, nil)

			res, err := h.ContactAddressDelete(ctx, tt.agent, tt.contactID, tt.addressID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}







func Test_ContactTagAdd(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID
		tagID     uuid.UUID

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			tagID:     uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.ContactTagAdd(ctx, tt.agent, tt.contactID, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ContactTagRemove(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		contactID uuid.UUID
		tagID     uuid.UUID

		responseContactGet *cmcontact.Contact
		responseContact    *cmcontact.Contact
		expectRes          *cmcontact.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			contactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
			tagID:     uuid.FromStringOrNil("d4e5f6a7-0003-11ec-0003-000000000003"),

			responseContactGet: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},
			expectRes: &cmcontact.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.ContactTagRemove(ctx, tt.agent, tt.contactID, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
