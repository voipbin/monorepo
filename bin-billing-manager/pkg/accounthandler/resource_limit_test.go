package accounthandler

import (
	"context"
	"testing"
	"time"

	commonbilling "monorepo/bin-common-handler/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_IsValidResourceLimit(t *testing.T) {

	type test struct {
		name string

		accountID    uuid.UUID
		resourceType commonbilling.ResourceType

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
			resourceType: commonbilling.ResourceTypeAgent,

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
			resourceType: commonbilling.ResourceTypeAgent,

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
			resourceType: commonbilling.ResourceTypeAgent,

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
			resourceType: commonbilling.ResourceTypeAgent,

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
			resourceType: commonbilling.ResourceTypeAgent,

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

			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

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
		resourceType commonbilling.ResourceType

		responseCount int

		expectRes int
	}

	customerID := uuid.FromStringOrNil("b1c2d3e4-0001-0001-0001-000000000001")

	tests := []test{
		{
			name:          "extension",
			customerID:    customerID,
			resourceType:  commonbilling.ResourceTypeExtension,
			responseCount: 3,
			expectRes:     3,
		},
		{
			name:          "trunk",
			customerID:    customerID,
			resourceType:  commonbilling.ResourceTypeTrunk,
			responseCount: 2,
			expectRes:     2,
		},
		{
			name:          "agent",
			customerID:    customerID,
			resourceType:  commonbilling.ResourceTypeAgent,
			responseCount: 5,
			expectRes:     5,
		},
		{
			name:          "queue",
			customerID:    customerID,
			resourceType:  commonbilling.ResourceTypeQueue,
			responseCount: 1,
			expectRes:     1,
		},
		{
			name:          "flow",
			customerID:    customerID,
			resourceType:  commonbilling.ResourceTypeFlow,
			responseCount: 4,
			expectRes:     4,
		},
		{
			name:          "conference",
			customerID:    customerID,
			resourceType:  commonbilling.ResourceTypeConference,
			responseCount: 2,
			expectRes:     2,
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
			case commonbilling.ResourceTypeExtension:
				mockReq.EXPECT().RegistrarV1ExtensionCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case commonbilling.ResourceTypeTrunk:
				mockReq.EXPECT().RegistrarV1TrunkCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case commonbilling.ResourceTypeAgent:
				mockReq.EXPECT().AgentV1AgentCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case commonbilling.ResourceTypeQueue:
				mockReq.EXPECT().QueueV1QueueCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case commonbilling.ResourceTypeFlow:
				mockReq.EXPECT().FlowV1FlowCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
			case commonbilling.ResourceTypeConference:
				mockReq.EXPECT().ConferenceV1ConferenceCountByCustomerID(ctx, tt.customerID).Return(tt.responseCount, nil)
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
	_, err := h.getResourceCount(ctx, customerID, commonbilling.ResourceType("unsupported"))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
