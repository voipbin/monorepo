package accounthandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_IsValidResourceLimit(t *testing.T) {

	type test struct {
		name string

		accountID    uuid.UUID
		resourceType account.ResourceType

		responseAccount *account.Account
		responseCount   int

		expectCountCall bool
		expectRes       bool
		expectErr       bool
	}

	tmDelete := time.Now()

	tests := []test{
		{
			name: "free plan - under limit",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000002"),
				},
				PlanType: account.PlanTypeFree,
			},
			responseCount: 3,

			expectCountCall: true,
			expectRes:       true,
			expectErr:       false,
		},
		{
			name: "free plan - at limit",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0002-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0002-0001-0001-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-0002-0001-0001-000000000002"),
				},
				PlanType: account.PlanTypeFree,
			},
			responseCount: 5,

			expectCountCall: true,
			expectRes:       false,
			expectErr:       false,
		},
		{
			name: "unlimited plan",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0003-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0003-0001-0001-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-0003-0001-0001-000000000002"),
				},
				PlanType: account.PlanTypeUnlimited,
			},

			expectCountCall: false,
			expectRes:       true,
			expectErr:       false,
		},
		{
			name: "deleted account",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0004-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0004-0001-0001-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-0004-0001-0001-000000000002"),
				},
				PlanType: account.PlanTypeFree,
				TMDelete: &tmDelete,
			},

			expectCountCall: false,
			expectRes:       false,
			expectErr:       false,
		},
		{
			name: "unknown plan type",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0005-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0005-0001-0001-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-0005-0001-0001-000000000002"),
				},
				PlanType: account.PlanType("invalid"),
			},

			expectCountCall: false,
			expectRes:       false,
			expectErr:       true,
		},
		{
			name: "db get error",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0006-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: nil,

			expectCountCall: false,
			expectRes:       false,
			expectErr:       true,
		},
		{
			name: "resource count error",

			accountID:    uuid.FromStringOrNil("a1b2c3d4-0007-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0007-0001-0001-000000000001"),
					CustomerID: uuid.FromStringOrNil("a1b2c3d4-0007-0001-0001-000000000002"),
				},
				PlanType: account.PlanTypeFree,
			},

			expectCountCall: false,
			expectRes:       false,
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			if tt.name == "db get error" {
				mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("database error"))

				_, err := h.IsValidResourceLimit(ctx, tt.accountID, tt.resourceType)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			if tt.name == "resource count error" {
				mockReq.EXPECT().AgentV1AgentCountByCustomerID(ctx, tt.responseAccount.CustomerID).Return(0, fmt.Errorf("count error"))

				_, err := h.IsValidResourceLimit(ctx, tt.accountID, tt.resourceType)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.expectCountCall {
				mockReq.EXPECT().AgentV1AgentCountByCustomerID(ctx, tt.responseAccount.CustomerID).Return(tt.responseCount, nil)
			}

			res, err := h.IsValidResourceLimit(ctx, tt.accountID, tt.resourceType)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getResourceCount(t *testing.T) {

	type test struct {
		name string

		customerID   uuid.UUID
		resourceType account.ResourceType

		responseCount int

		expectRes int
	}

	customerID := uuid.FromStringOrNil("b1c2d3e4-0001-0001-0001-000000000001")

	tests := []test{
		{
			name:          "extension",
			customerID:    customerID,
			resourceType:  account.ResourceTypeExtension,
			responseCount: 3,
			expectRes:     3,
		},
		{
			name:          "trunk",
			customerID:    customerID,
			resourceType:  account.ResourceTypeTrunk,
			responseCount: 2,
			expectRes:     2,
		},
		{
			name:          "agent",
			customerID:    customerID,
			resourceType:  account.ResourceTypeAgent,
			responseCount: 5,
			expectRes:     5,
		},
		{
			name:          "queue",
			customerID:    customerID,
			resourceType:  account.ResourceTypeQueue,
			responseCount: 1,
			expectRes:     1,
		},
		{
			name:          "flow",
			customerID:    customerID,
			resourceType:  account.ResourceTypeFlow,
			responseCount: 4,
			expectRes:     4,
		},
		{
			name:          "conference",
			customerID:    customerID,
			resourceType:  account.ResourceTypeConference,
			responseCount: 2,
			expectRes:     2,
		},
		{
			name:          "virtual_number",
			customerID:    customerID,
			resourceType:  account.ResourceTypeVirtualNumber,
			responseCount: 3,
			expectRes:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			switch tt.resourceType {
			case account.ResourceTypeExtension:
				mockReq.EXPECT().RegistrarV1ExtensionCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case account.ResourceTypeTrunk:
				mockReq.EXPECT().RegistrarV1TrunkCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case account.ResourceTypeAgent:
				mockReq.EXPECT().AgentV1AgentCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case account.ResourceTypeQueue:
				mockReq.EXPECT().QueueV1QueueCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case account.ResourceTypeFlow:
				mockReq.EXPECT().FlowV1FlowCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case account.ResourceTypeConference:
				mockReq.EXPECT().ConferenceV1ConferenceCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case account.ResourceTypeVirtualNumber:
				mockReq.EXPECT().NumberV1VirtualNumberCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			}

			res, err := h.getResourceCount(ctx, tt.customerID, tt.resourceType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getResourceCount_unsupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := accountHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("b1c2d3e4-0002-0001-0001-000000000001")
	_, err := h.getResourceCount(ctx, customerID, account.ResourceType("unsupported"))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_IsValidResourceLimitByCustomerID(t *testing.T) {

	type test struct {
		name string

		customerID   uuid.UUID
		resourceType account.ResourceType

		responseCustomer *cmcustomer.Customer
		responseAccount  *account.Account
		responseCount    int

		expectCountCall bool
		expectRes       bool
		expectErr       bool
	}

	tests := []test{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("c1d1e1f1-0001-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("c1d1e1f1-0001-0001-0001-000000000001"),
				BillingAccountID: uuid.FromStringOrNil("c1d1e1f1-0001-0001-0001-000000000002"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1d1e1f1-0001-0001-0001-000000000002"),
					CustomerID: uuid.FromStringOrNil("c1d1e1f1-0001-0001-0001-000000000001"),
				},
				PlanType: account.PlanTypeFree,
			},
			responseCount: 3,

			expectCountCall: true,
			expectRes:       true,
			expectErr:       false,
		},
		{
			name: "customer not found error",

			customerID:   uuid.FromStringOrNil("c2d2e2f2-0002-0001-0001-000000000001"),
			resourceType: account.ResourceTypeAgent,

			responseCustomer: nil,
			responseAccount:  nil,

			expectCountCall: false,
			expectRes:       false,
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			if tt.name == "customer not found error" {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(nil, fmt.Errorf("customer not found"))

				_, err := h.IsValidResourceLimitByCustomerID(ctx, tt.customerID, tt.resourceType)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			// GetByCustomerID calls AccountGet
			mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)
			// IsValidResourceLimit calls h.Get which calls AccountGet again
			mockDB.EXPECT().AccountGet(ctx, tt.responseAccount.ID).Return(tt.responseAccount, nil)

			if tt.expectCountCall {
				mockReq.EXPECT().AgentV1AgentCountByCustomerID(ctx, tt.responseAccount.CustomerID).Return(tt.responseCount, nil)
			}

			res, err := h.IsValidResourceLimitByCustomerID(ctx, tt.customerID, tt.resourceType)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
