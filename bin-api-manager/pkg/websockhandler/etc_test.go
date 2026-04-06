package websockhandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	commonidentity "monorepo/bin-common-handler/models/identity"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_validateTopics(t *testing.T) {
	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		topics []string

		expectRes bool
	}{
		{
			name: "super admin doesn't need any validation",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			topics: []string{
				"some_invalid_topic",
			},

			expectRes: true,
		},
		{
			name: "all topics are valid",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8107ec3c-da97-11ee-8712-2b7a730c795e"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			topics: []string{
				"agent_id:8107ec3c-da97-11ee-8712-2b7a730c795e:chat",
				"agent_id:8107ec3c-da97-11ee-8712-2b7a730c795e:chatroom",
			},

			expectRes: true,
		},
		{
			name: "one topic is invalid",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a53950e6-da97-11ee-b4cf-3f9ab730fa90"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			topics: []string{
				"agent_id:a53950e6-da97-11ee-b4cf-3f9ab730fa90:chat",
				"invalid topic name",
			},

			expectRes: false,
		},
		{
			name: "direct token with valid 4-part topics",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				ResourceType:         dmdirect.ResourceTypeAI,
				ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			topics: []string{
				"customer_id:5f84c116-da29-11ee-b479-a70bca2a0a48:aicall:e5f6a7b8-0000-0000-0000-000000000000",
			},

			expectRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := websockHandler{}
			ctx := context.Background()

			res := h.validateTopics(ctx, tt.agent, tt.topics)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_validateTopic(t *testing.T) {
	tests := []struct {
		name string

		agent *auth.AuthIdentity
		topic string

		expectRes bool
	}{
		{
			name: "agent has agent permission and subscribe to the agent level topic",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			topic: "agent_id:a78226f4-da95-11ee-a9fa-6f64fb4b7018:chatroom",

			expectRes: true,
		},
		{
			name: "agent has agent permission but subscribe to the wrong agent id",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			topic: "agent_id:59141620-da96-11ee-a825-3737b4635c9c:chatroom",

			expectRes: false,
		},
		{
			name: "agent has agent permission but subscribe to the customer level topic",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			topic: "customer_id:5f84c116-da29-11ee-b479-a70bca2a0a48:chatroom",

			expectRes: false,
		},
		{
			name: "agent has admin permission but subscribe to the wrong agent id",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			topic: "agent_id:59141620-da96-11ee-a825-3737b4635c9c:chatroom",

			expectRes: false,
		},
		{
			name: "agent has admin permission and subscribe to the customer level topic",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			topic: "customer_id:5f84c116-da29-11ee-b479-a70bca2a0a48:chatroom",

			expectRes: true,
		},
		{
			name: "agent has admin permission and subscribe to the agent level topic",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			topic: "agent_id:a78226f4-da95-11ee-a9fa-6f64fb4b7018:chatroom",

			expectRes: true,
		},
		{
			name: "agent has admin permission but subscribe to the wrong customer id",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a78226f4-da95-11ee-a9fa-6f64fb4b7018"),
					CustomerID: uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			topic: "customer_id:0c719e72-da97-11ee-baf5-838bed050454:chatroom",

			expectRes: false,
		},
		{
			name: "direct token can subscribe to scoped 4-part topic",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				ResourceType:         dmdirect.ResourceTypeAI,
				ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			topic: "customer_id:5f84c116-da29-11ee-b479-a70bca2a0a48:aicall:e5f6a7b8-0000-0000-0000-000000000000",

			expectRes: true,
		},
		{
			name: "direct token cannot subscribe to 2-part customer topic",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				ResourceType:         dmdirect.ResourceTypeAI,
				ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			topic: "customer_id:5f84c116-da29-11ee-b479-a70bca2a0a48",

			expectRes: false,
		},
		{
			name: "direct token cannot subscribe to disallowed resource type",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				ResourceType:         dmdirect.ResourceTypeAI,
				ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			topic: "customer_id:5f84c116-da29-11ee-b479-a70bca2a0a48:call:e5f6a7b8-0000-0000-0000-000000000000",

			expectRes: false,
		},
		{
			name: "direct token cannot subscribe to wrong customer id",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f84c116-da29-11ee-b479-a70bca2a0a48"),
				ResourceType:         dmdirect.ResourceTypeAI,
				ResourceID:           uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			topic: "customer_id:0c719e72-da97-11ee-baf5-838bed050454:aicall:e5f6a7b8-0000-0000-0000-000000000000",

			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := websockHandler{}
			ctx := context.Background()

			res := h.validateTopic(ctx, tt.agent, tt.topic)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
