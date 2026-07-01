package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ContactAddressCreateIndependent_Unresolved(t *testing.T) {

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseAddress := &cmcontact.Address{
		ID:         uuid.FromStringOrNil("2c9c9f0a-5066-11ec-ab34-23643cfdc1c5"),
		CustomerID: customerID,
		ContactID:  uuid.Nil,
		Type:       "tel",
		Target:     "+821100000001",
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		reqHandler: mockReq,
	}
	ctx := context.Background()

	// contactID == uuid.Nil must NOT trigger a contact lookup, and must pass
	// uuid.Nil straight through to the create RPC, scoped by a.CustomerID.
	mockReq.EXPECT().ContactV1ContactAddressCreate(ctx, customerID, uuid.Nil, "tel", "+821100000001", false, "", "").Return(responseAddress, nil)

	res, err := h.ContactAddressCreateIndependent(ctx, a, uuid.Nil, "tel", "+821100000001", false, "", "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, responseAddress) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", responseAddress, res)
	}
}

func Test_ContactAddressCreateIndependent_Unresolved_IsPrimaryRejectedUpstream(t *testing.T) {
	// The HTTP-layer 400 guard for "unresolved + is_primary" lives in server/,
	// not the servicehandler. This test only documents that the servicehandler
	// itself does not special-case isPrimary — it passes whatever it is given
	// straight through, matching the resolved-creation path's behavior.
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseAddress := &cmcontact.Address{
		ID:         uuid.FromStringOrNil("2c9c9f0a-5066-11ec-ab34-23643cfdc1c5"),
		CustomerID: customerID,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		reqHandler: mockReq,
	}
	ctx := context.Background()

	mockReq.EXPECT().ContactV1ContactAddressCreate(ctx, customerID, uuid.Nil, "tel", "+821100000001", true, "", "").Return(responseAddress, nil)

	_, err := h.ContactAddressCreateIndependent(ctx, a, uuid.Nil, "tel", "+821100000001", true, "", "")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ContactAddressClaim(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		addressID uuid.UUID
		contactID uuid.UUID

		responseAddress *cmcontact.Address
		responseContact *cmcontact.Contact
		responseClaimed *cmcontact.Address

		expectAddressGetCall bool
		expectContactGetCall bool
		expectClaimCall      bool
		expectErr            error
		expectRes            *cmcontact.Address
	}

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID := uuid.FromStringOrNil("9d3a7fbe-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	addressID := uuid.FromStringOrNil("2c9c9f0a-5066-11ec-ab34-23643cfdc1c5")
	contactID := uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	tests := []test{
		{
			name: "normal - happy path claims successfully",

			agent:     agent,
			addressID: addressID,
			contactID: contactID,

			responseAddress: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
				ContactID:  uuid.Nil,
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
			},
			responseClaimed: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
				ContactID:  contactID,
			},

			expectAddressGetCall: true,
			expectContactGetCall: true,
			expectClaimCall:      true,
			expectRes: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
				ContactID:  contactID,
			},
		},
		{
			name: "cross-tenant address rejected",

			agent:     agent,
			addressID: addressID,
			contactID: contactID,

			responseAddress: &cmcontact.Address{
				ID:         addressID,
				CustomerID: otherCustomerID,
			},

			expectAddressGetCall: true,
			expectContactGetCall: false,
			expectClaimCall:      false,
			expectErr:            serviceerrors.ErrNotFound,
		},
		{
			name: "cross-tenant contact rejected",

			agent:     agent,
			addressID: addressID,
			contactID: contactID,

			responseAddress: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{ID: contactID, CustomerID: otherCustomerID},
			},

			expectAddressGetCall: true,
			expectContactGetCall: true,
			expectClaimCall:      false,
			expectErr:            serviceerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			if tt.expectAddressGetCall {
				mockReq.EXPECT().ContactV1ContactAddressGet(ctx, tt.agent.CustomerID, tt.addressID).Return(tt.responseAddress, nil)
			}
			if tt.expectContactGetCall {
				mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			}
			if tt.expectClaimCall {
				mockReq.EXPECT().ContactV1ContactAddressClaim(ctx, tt.agent.CustomerID, tt.addressID, tt.contactID).Return(tt.responseClaimed, nil)
			}

			res, err := h.ContactAddressClaim(ctx, tt.agent, tt.addressID, tt.contactID)
			if tt.expectErr != nil {
				if err != tt.expectErr {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentContactAddressClaim mirrors Test_ContactAddressClaim's
// coverage for the service-agent surface — happy path plus both cross-tenant
// rejection branches, asserting they return serviceerrors.ErrNotFound (404),
// not ErrPermissionDenied (403), per the design doc's anti-enumeration
// decision.
func Test_ServiceAgentContactAddressClaim(t *testing.T) {

	type test struct {
		name string

		agent     *auth.AuthIdentity
		addressID uuid.UUID
		contactID uuid.UUID

		responseAgent   *amagent.Agent
		responseAddress *cmcontact.Address
		responseContact *cmcontact.Contact
		responseClaimed *cmcontact.Address

		expectContactGetCall bool
		expectClaimCall      bool
		expectErr            error
		expectRes            *cmcontact.Address
	}

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID := uuid.FromStringOrNil("9d3a7fbe-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	addressID := uuid.FromStringOrNil("2c9c9f0a-5066-11ec-ab34-23643cfdc1c5")
	contactID := uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseAgent := &amagent.Agent{
		Identity: commonidentity.Identity{ID: agentID, CustomerID: customerID},
	}

	tests := []test{
		{
			name: "normal - happy path claims successfully",

			agent:     agent,
			addressID: addressID,
			contactID: contactID,

			responseAgent: responseAgent,
			responseAddress: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
				ContactID:  uuid.Nil,
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{ID: contactID, CustomerID: customerID},
			},
			responseClaimed: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
				ContactID:  contactID,
			},

			expectContactGetCall: true,
			expectClaimCall:      true,
			expectRes: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
				ContactID:  contactID,
			},
		},
		{
			name: "cross-tenant address rejected",

			agent:     agent,
			addressID: addressID,
			contactID: contactID,

			responseAgent: responseAgent,
			responseAddress: &cmcontact.Address{
				ID:         addressID,
				CustomerID: otherCustomerID,
			},

			expectContactGetCall: false,
			expectClaimCall:      false,
			expectErr:            serviceerrors.ErrNotFound,
		},
		{
			name: "cross-tenant contact rejected",

			agent:     agent,
			addressID: addressID,
			contactID: contactID,

			responseAgent: responseAgent,
			responseAddress: &cmcontact.Address{
				ID:         addressID,
				CustomerID: customerID,
			},
			responseContact: &cmcontact.Contact{
				Identity: commonidentity.Identity{ID: contactID, CustomerID: otherCustomerID},
			},

			expectContactGetCall: true,
			expectClaimCall:      false,
			expectErr:            serviceerrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agent.AgentID()).Return(tt.responseAgent, nil)
			mockReq.EXPECT().ContactV1ContactAddressGet(ctx, tt.responseAgent.CustomerID, tt.addressID).Return(tt.responseAddress, nil)
			if tt.expectContactGetCall {
				mockReq.EXPECT().ContactV1ContactGet(ctx, tt.contactID).Return(tt.responseContact, nil)
			}
			if tt.expectClaimCall {
				mockReq.EXPECT().ContactV1ContactAddressClaim(ctx, tt.responseAgent.CustomerID, tt.addressID, tt.contactID).Return(tt.responseClaimed, nil)
			}

			res, err := h.ServiceAgentContactAddressClaim(ctx, tt.agent, tt.addressID, tt.contactID)
			if tt.expectErr != nil {
				if err != tt.expectErr {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
