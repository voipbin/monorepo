package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ServiceAgentMeGet(t *testing.T) {

	tests := []struct {
		name string

		agent *auth.AuthIdentity

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			}),

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.agent.AgentID()).Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeGet(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdate(t *testing.T) {

	tests := []struct {
		name string

		agent      *auth.AuthIdentity
		agentName  *string
		detail     *string
		ringMethod *amagent.RingMethod

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			}),
			agentName:  stringPtr("update name"),
			detail:     stringPtr("update detail"),
			ringMethod: ringMethodPtr(amagent.RingMethodRingAll),

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdate(ctx, tt.agent.AgentID(), tt.agentName, tt.detail, tt.ringMethod).Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdate(ctx, tt.agent, tt.agentName, tt.detail, tt.ringMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdateAddresses(t *testing.T) {

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		addresses []commonaddress.Address

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			}),
			addresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+123456789",
				},
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdateAddresses(ctx, tt.agent.AgentID(), tt.addresses).Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdateAddresses(ctx, tt.agent, tt.addresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		status amagent.Status

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			}),
			status: amagent.StatusAvailable,

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdateStatus(ctx, tt.agent.AgentID(), tt.status).Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdateStatus(ctx, tt.agent, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdatePassword(t *testing.T) {

	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		password string

		responseAgent *amagent.Agent
		expectedRes   *amagent.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			}),
			password: "update_password",

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},
			expectedRes: &amagent.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
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

			mockReq.EXPECT().AgentV1AgentUpdatePassword(ctx, gomock.Any(), tt.agent.AgentID(), tt.password).Return(tt.responseAgent, nil)

			res, err := h.ServiceAgentMeUpdatePassword(ctx, tt.agent, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}

func Test_ServiceAgentMeUpdateAddresses_extension(t *testing.T) {
	agentID := uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03")
	ownExtID := "49b41028-2d8d-11ef-b38d-27dd55f2bb71"
	otherExtID := "8a0f4bb2-2d8d-11ef-9c5a-1b2c3d4e5f60"

	ownExt := commonaddress.Address{Type: commonaddress.TypeExtension, Target: ownExtID, TargetName: "1001", Name: "desk", Detail: "d"}
	tel := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+123****6789"}

	newAgent := func(permission amagent.Permission) *auth.AuthIdentity {
		return auth.NewAgentIdentity(&amagent.Agent{
			Identity:   commonidentity.Identity{ID: agentID},
			Permission: permission,
		})
	}

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		addresses []commonaddress.Address
		stored    []commonaddress.Address

		expectLookup bool // the agent record is read to check the extensions
		expectUpdate bool // the request is forwarded to agent-manager
	}{
		{
			name:         "a tel and sip only request does not read the agent record",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{tel, {Type: commonaddress.TypeSIP, Target: "alice@example.com"}},
			expectUpdate: true,
		},
		{
			name:         "an empty request is forwarded (removes everything)",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{},
			expectUpdate: true,
		},
		{
			name:         "the stored extension sent back unchanged is retained",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{ownExt, tel},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
			expectUpdate: true,
		},
		{
			name:         "the order of the addresses does not matter",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{tel, ownExt},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
			expectUpdate: true,
		},
		{
			name:         "the name and detail of a stored extension can change",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: ownExtID, TargetName: "1001", Name: "renamed"}},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
			expectUpdate: true,
		},
		{
			name:         "removing the stored extension while keeping the tel is allowed",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{tel},
			stored:       []commonaddress.Address{ownExt},
			expectUpdate: true,
		},
		{
			name:         "adding an extension that the agent does not have is rejected",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: otherExtID, TargetName: "1002"}},
			stored:       []commonaddress.Address{},
			expectLookup: true,
		},
		{
			name:         "adding a second extension next to the stored one is rejected",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{ownExt, {Type: commonaddress.TypeExtension, Target: otherExtID, TargetName: "1002"}},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
		},
		{
			name:         "changing the target of the stored extension is rejected",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: otherExtID, TargetName: "1001"}},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
		},
		{
			name:         "changing the target name of the stored extension is rejected",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: ownExtID, TargetName: "9999"}},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
		},
		{
			name:         "a variant form of a stored canonical extension is rejected",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: strings.ToUpper(ownExtID), TargetName: "1001"}},
			stored:       []commonaddress.Address{ownExt},
			expectLookup: true,
		},
		{
			name:         "a malformed extension target is rejected",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: "not-a-uuid"}},
			stored:       []commonaddress.Address{},
			expectLookup: true,
		},
		{
			name:         "a stored non-canonical extension can not be retained by sending it back",
			agent:        newAgent(amagent.PermissionCustomerAgent),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: strings.ToUpper(ownExtID), TargetName: "1001"}},
			stored:       []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: strings.ToUpper(ownExtID), TargetName: "1001"}},
			expectLookup: true,
		},
		{
			name:         "the role of the caller is not evaluated (admin is rejected as well)",
			agent:        newAgent(amagent.PermissionCustomerAdmin),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: otherExtID, TargetName: "1002"}},
			stored:       []commonaddress.Address{},
			expectLookup: true,
		},
		{
			name:         "the role of the caller is not evaluated (manager is rejected as well)",
			agent:        newAgent(amagent.PermissionCustomerManager),
			addresses:    []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: otherExtID, TargetName: "1002"}},
			stored:       []commonaddress.Address{},
			expectLookup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &serviceHandler{reqHandler: mockReq}
			ctx := context.Background()

			if tt.expectLookup {
				mockReq.EXPECT().AgentV1AgentGet(ctx, agentID).Return(&amagent.Agent{
					Identity:  commonidentity.Identity{ID: agentID},
					Addresses: tt.stored,
				}, nil).Times(1)
			}
			if tt.expectUpdate {
				mockReq.EXPECT().AgentV1AgentUpdateAddresses(ctx, agentID, tt.addresses).Return(&amagent.Agent{Identity: commonidentity.Identity{ID: agentID}}, nil).Times(1)
			}

			res, err := h.ServiceAgentMeUpdateAddresses(ctx, tt.agent, tt.addresses)

			if tt.expectUpdate {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}

			if res != nil {
				t.Errorf("Wrong match. expect: no result, got: %v", res)
			}
			var ve *cerrors.VoipbinError
			if !errors.As(err, &ve) {
				t.Fatalf("Wrong match. expect: a typed VoipbinError, got: %v", err)
			}
			if ve.Status != cerrors.StatusPermissionDenied || ve.Reason != "EXTENSION_ADDRESS_ADMIN_ONLY" || ve.Domain != string(commonoutline.ServiceNameAPIManager) {
				t.Errorf("Wrong match. expect: PERMISSION_DENIED/EXTENSION_ADDRESS_ADMIN_ONLY from api-manager, got: %+v", ve)
			}
		})
	}
}

func Test_ServiceAgentMeUpdateAddresses_extension_lookup_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	agentID := uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03")
	a := auth.NewAgentIdentity(&amagent.Agent{Identity: commonidentity.Identity{ID: agentID}})

	// the update must not be forwarded when the stored extensions can not be verified (fail closed).
	mockReq.EXPECT().AgentV1AgentGet(ctx, agentID).Return(nil, errors.New("rpc failed"))

	res, err := h.ServiceAgentMeUpdateAddresses(ctx, a, []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: "49b41028-2d8d-11ef-b38d-27dd55f2bb71"}})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
	}
}
