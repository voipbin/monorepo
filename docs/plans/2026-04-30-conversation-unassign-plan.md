# Conversation Unassign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add explicit unassign endpoints for conversations on both the admin (`/conversations`) and agent (`/service_agents/conversations`) API surfaces, and remove the complex owning-agent carve-out from `PUT /conversations/<id>`.

**Architecture:** Four HTTP surface changes confined entirely to `bin-openapi-manager` (OpenAPI spec) and `bin-api-manager` (server + servicehandler layers). No conversation-manager changes — all four endpoints ultimately call the existing `ConversationV1ConversationUpdate` RPC with `{FieldOwnerID: uuid.Nil}`.

**Tech Stack:** Go, oapi-codegen, gomock, gin, RabbitMQ RPC via `requesthandler`.

---

## Task 1: Add OpenAPI path files

**Files:**
- Create: `bin-openapi-manager/openapi/paths/conversations/id_unassign.yaml`
- Modify: `bin-openapi-manager/openapi/paths/service_agents/conversations_id.yaml`
- Create: `bin-openapi-manager/openapi/paths/service_agents/conversations_id_unassign.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Create `conversations/id_unassign.yaml`**

```yaml
post:
  summary: Unassign the conversation
  description: |
    Removes the current owner from the conversation. Admin and manager callers may unassign any
    conversation. The owning agent may unassign themselves. Returns 403 if the caller is neither
    an admin/manager nor the current owner.
  tags:
    - Conversation
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
        example: "828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"
      description: "The unique identifier of the conversation. Returned from the `GET /conversations` response."
  responses:
    '200':
      description: Conversation after unassignment.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ConversationManagerConversation'
    '400':
      $ref: '#/components/responses/BadRequest'
    '401':
      $ref: '#/components/responses/Unauthenticated'
    '403':
      $ref: '#/components/responses/PermissionDenied'
    '404':
      $ref: '#/components/responses/NotFound'
    '500':
      $ref: '#/components/responses/InternalError'
```

**Step 2: Add `put:` block to `service_agents/conversations_id.yaml`**

Open `bin-openapi-manager/openapi/paths/service_agents/conversations_id.yaml`. It currently only has `get:`. Append after the closing of `get:`:

```yaml
put:
  summary: Update conversation info
  description: Updates the details of a specific conversation by its ID. Admin and manager callers only.
  tags:
    - Service Agent
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
        example: "828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"
      description: "The unique identifier of the conversation. Returned from the `GET /service_agents/conversations` response."
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            owner_type:
              type: string
              example: "agent"
            owner_id:
              type: string
              format: uuid
              example: "d152e69e-105b-11ee-b395-eb18426de979"
            name:
              type: string
              example: "VIP customer"
            detail:
              type: string
              example: "Preferred language: Korean"
  responses:
    '200':
      description: The updated conversation.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ConversationManagerConversation'
    '400':
      $ref: '#/components/responses/BadRequest'
    '401':
      $ref: '#/components/responses/Unauthenticated'
    '403':
      $ref: '#/components/responses/PermissionDenied'
    '404':
      $ref: '#/components/responses/NotFound'
    '500':
      $ref: '#/components/responses/InternalError'
```

**Step 3: Create `service_agents/conversations_id_unassign.yaml`**

```yaml
post:
  summary: Unassign the conversation
  description: |
    Removes the current owner from the conversation. Admin and manager callers may unassign any
    conversation. The owning agent may unassign themselves. Returns 403 if the caller is neither
    an admin/manager nor the current owner.
  tags:
    - Service Agent
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
        example: "828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"
      description: "The unique identifier of the conversation. Returned from the `GET /service_agents/conversations` response."
  responses:
    '200':
      description: Conversation after unassignment.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ConversationManagerConversation'
    '400':
      $ref: '#/components/responses/BadRequest'
    '401':
      $ref: '#/components/responses/Unauthenticated'
    '403':
      $ref: '#/components/responses/PermissionDenied'
    '404':
      $ref: '#/components/responses/NotFound'
    '500':
      $ref: '#/components/responses/InternalError'
```

**Step 4: Register new paths in `openapi.yaml`**

Find the block in `openapi.yaml` that contains:
```yaml
  /v1.0/conversations/{id}:
    $ref: './paths/conversations/id.yaml'
```

Add after it:
```yaml
  /v1.0/conversations/{id}/unassign:
    $ref: './paths/conversations/id_unassign.yaml'
```

Find the block that contains:
```yaml
  /v1.0/service_agents/conversations/{id}:
    $ref: './paths/service_agents/conversations_id.yaml'
```

Add after it:
```yaml
  /v1.0/service_agents/conversations/{id}/unassign:
    $ref: './paths/service_agents/conversations_id_unassign.yaml'
```

**Step 5: Run codegen in bin-openapi-manager**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./...
```

Expected: `gens/models/gen.go` regenerated without errors.

**Step 6: Commit**

```bash
git add bin-openapi-manager/
git commit -m "NOJIRA-Conversation-unassign-design

- bin-openapi-manager: Add POST /conversations/{id}/unassign path
- bin-openapi-manager: Add PUT /service_agents/conversations/{id} path
- bin-openapi-manager: Add POST /service_agents/conversations/{id}/unassign path"
```

---

## Task 2: Regenerate api-manager generated code

**Files:**
- Modify: `bin-api-manager/gens/openapi_server/gen.go` (auto-generated)

**Step 1: Run codegen in bin-api-manager**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

Expected: `gens/openapi_server/gen.go` now contains new method stubs:
- `PostConversationsIdUnassign`
- `PutServiceAgentsConversationsId`
- `PostServiceAgentsConversationsIdUnassign`

Verify:
```bash
grep -n "PostConversationsIdUnassign\|PutServiceAgentsConversationsId\|PostServiceAgentsConversationsIdUnassign" bin-api-manager/gens/openapi_server/gen.go
```

Expected: 3 matches found.

---

## Task 3: Add interface methods to servicehandler

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go`

**Step 1: Locate the conversation handler interface block**

Open `bin-api-manager/pkg/servicehandler/main.go`. Find the comment `// conversation handlers` (around line 427). It lists `ConversationGet`, `ConversationGetsByCustomerID`, `ConversationUpdate`, etc.

**Step 2: Add three new method signatures**

After `ConversationUpdate(...)`, add:

```go
ConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
```

Find the service-agents conversation block (around line 777). It lists `ServiceAgentConversationGet`, `ServiceAgentConversationList`, etc. After `ServiceAgentConversationList(...)`, add:

```go
ServiceAgentConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error)
ServiceAgentConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
```

**Step 3: Regenerate mock**

```bash
cd bin-api-manager
go generate ./...
```

Expected: `pkg/servicehandler/mock_main.go` now includes mock implementations of the three new methods. Verify:

```bash
grep -n "ConversationUnassign\|ServiceAgentConversationUpdate\|ServiceAgentConversationUnassign" bin-api-manager/pkg/servicehandler/mock_main.go
```

Expected: 3+ matches.

---

## Task 4: Simplify ConversationUpdate — remove agent carve-out

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/conversation.go`
- Modify: `bin-api-manager/pkg/servicehandler/conversation_test.go`

**Step 1: Write the updated (failing) test first**

In `conversation_test.go`, find `Test_ConversationUpdate_PermissionDenied`. Add a new test case for "owning agent calls PUT — now denied":

```go
{
    name: "owning agent calls PUT — now rejected (breaking change)",
    agent: auth.NewAgentIdentity(&amagent.Agent{
        Identity: commonidentity.Identity{
            ID:         owningAgentID,
            CustomerID: customerID,
        },
        Permission: amagent.PermissionNone,
    }),
    conversationID: conversationID,
    fields: map[cvconversation.Field]any{
        cvconversation.FieldOwnerID: uuid.Nil,
    },
    responseConversation: &cvconversation.Conversation{
        Identity: commonidentity.Identity{
            ID:         conversationID,
            CustomerID: customerID,
        },
        Owner: commonidentity.Owner{
            OwnerType: commonidentity.OwnerTypeAgent,
            OwnerID:   owningAgentID,
        },
    },
},
```

**Step 2: Run test to verify it currently PASSES (i.e., agent self-unassign still works)**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -run Test_ConversationUpdate_PermissionDenied -v
```

Expected: The new case FAILS (it expects PermissionDenied but currently gets nil error because the carve-out allows it). This confirms the test is detecting the old behaviour correctly.

**Step 3: Remove the "owning agent self-unassigns" success case from Test_ConversationUpdate**

In `Test_ConversationUpdate`, delete the entire test case struct with `name: "owning agent self-unassigns owner_id (nil UUID)"` (lines ~476–510).

**Step 4: Simplify ConversationUpdate in conversation.go**

Replace the current permission block (lines ~175–188):

```go
// REMOVE this entire block:
if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
    if !a.IsAgent() || a.Agent == nil {
        log.Info("Caller is not an agent.")
        return nil, serviceerrors.ErrPermissionDenied
    }
    if c.OwnerType != commonidentity.OwnerTypeAgent || c.OwnerID != a.Agent.ID {
        log.Info("Caller is not the owning agent.")
        return nil, serviceerrors.ErrPermissionDenied
    }
    if !payloadIsExactlySelfUnassign(fields) {
        log.Info("Non-admin agent payload is not exactly a self-unassign.")
        return nil, serviceerrors.ErrPermissionDenied
    }
}
```

Replace with:

```go
if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
    log.Info("Caller has no permission to update the conversation.")
    return nil, serviceerrors.ErrPermissionDenied
}
```

**Step 5: Delete payloadIsExactlySelfUnassign**

Delete the entire `payloadIsExactlySelfUnassign` function (lines ~201–222 in the current file).

**Step 6: Run tests**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -run "Test_ConversationUpdate" -v
```

Expected: All cases in `Test_ConversationUpdate` pass. The new PermissionDenied case for owning agent now passes too.

**Step 7: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/conversation.go bin-api-manager/pkg/servicehandler/conversation_test.go
git commit -m "NOJIRA-Conversation-unassign-design

- bin-api-manager: Remove owning-agent self-unassign carve-out from ConversationUpdate
- bin-api-manager: Delete payloadIsExactlySelfUnassign helper"
```

---

## Task 5: Add ConversationUnassign method and tests

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/conversation.go`
- Modify: `bin-api-manager/pkg/servicehandler/conversation_test.go`

**Step 1: Write the failing tests first**

Add a new test function at the bottom of `conversation_test.go`:

```go
func Test_ConversationUnassign(t *testing.T) {
    customerID    := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
    conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
    owningAgentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
    otherAgentID  := uuid.FromStringOrNil("a01f2c3a-3001-11f0-9d11-2bd5b4a45af1")

    assignedConversation := &cvconversation.Conversation{
        Identity: commonidentity.Identity{
            ID:         conversationID,
            CustomerID: customerID,
        },
        Owner: commonidentity.Owner{
            OwnerType: commonidentity.OwnerTypeAgent,
            OwnerID:   owningAgentID,
        },
    }
    unassignedConversation := &cvconversation.Conversation{
        Identity: commonidentity.Identity{
            ID:         conversationID,
            CustomerID: customerID,
        },
    }

    unassignFields := map[cvconversation.Field]any{
        cvconversation.FieldOwnerID: uuid.Nil,
    }

    type test struct {
        name           string
        agent          *auth.AuthIdentity
        conversationID uuid.UUID

        // set responseConversation to nil to skip the ConversationV1ConversationGet mock
        responseConversation *cvconversation.Conversation
        // set expectCallUpdate to true if ConversationV1ConversationUpdate should be called
        expectCallUpdate bool
        responseUpdated  *cvconversation.Conversation

        expectErr error
        expectRes *cvconversation.WebhookMessage
    }

    tests := []test{
        {
            name: "admin unassigns assigned conversation",
            agent: auth.NewAgentIdentity(&amagent.Agent{
                Identity: commonidentity.Identity{
                    ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
                    CustomerID: customerID,
                },
                Permission: amagent.PermissionCustomerAdmin,
            }),
            conversationID:       conversationID,
            responseConversation: assignedConversation,
            expectCallUpdate:     true,
            responseUpdated:      unassignedConversation,
            expectRes: &cvconversation.WebhookMessage{
                Identity: commonidentity.Identity{
                    ID:         conversationID,
                    CustomerID: customerID,
                },
            },
        },
        {
            name: "manager unassigns assigned conversation",
            agent: auth.NewAgentIdentity(&amagent.Agent{
                Identity: commonidentity.Identity{
                    ID:         uuid.FromStringOrNil("d6a401d4-3002-11f0-9c79-b3a64c98c8d9"),
                    CustomerID: customerID,
                },
                Permission: amagent.PermissionCustomerManager,
            }),
            conversationID:       conversationID,
            responseConversation: assignedConversation,
            expectCallUpdate:     true,
            responseUpdated:      unassignedConversation,
            expectRes: &cvconversation.WebhookMessage{
                Identity: commonidentity.Identity{
                    ID:         conversationID,
                    CustomerID: customerID,
                },
            },
        },
        {
            name: "owning agent self-unassigns",
            agent: auth.NewAgentIdentity(&amagent.Agent{
                Identity: commonidentity.Identity{
                    ID:         owningAgentID,
                    CustomerID: customerID,
                },
                Permission: amagent.PermissionNone,
            }),
            conversationID:       conversationID,
            responseConversation: assignedConversation,
            expectCallUpdate:     true,
            responseUpdated:      unassignedConversation,
            expectRes: &cvconversation.WebhookMessage{
                Identity: commonidentity.Identity{
                    ID:         conversationID,
                    CustomerID: customerID,
                },
            },
        },
        {
            name: "non-owning agent — permission denied",
            agent: auth.NewAgentIdentity(&amagent.Agent{
                Identity: commonidentity.Identity{
                    ID:         otherAgentID,
                    CustomerID: customerID,
                },
                Permission: amagent.PermissionNone,
            }),
            conversationID:       conversationID,
            responseConversation: assignedConversation,
            expectCallUpdate:     false,
            expectErr:            serviceerrors.ErrPermissionDenied,
        },
        {
            name: "owning agent calls unassign on already-unassigned conversation — permission denied",
            agent: auth.NewAgentIdentity(&amagent.Agent{
                Identity: commonidentity.Identity{
                    ID:         owningAgentID,
                    CustomerID: customerID,
                },
                Permission: amagent.PermissionNone,
            }),
            conversationID:       conversationID,
            responseConversation: unassignedConversation, // owner_id == uuid.Nil
            expectCallUpdate:     false,
            expectErr:            serviceerrors.ErrPermissionDenied,
        },
        {
            name: "admin calls unassign on already-unassigned conversation — idempotent 200",
            agent: auth.NewAgentIdentity(&amagent.Agent{
                Identity: commonidentity.Identity{
                    ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
                    CustomerID: customerID,
                },
                Permission: amagent.PermissionCustomerAdmin,
            }),
            conversationID:       conversationID,
            responseConversation: unassignedConversation,
            expectCallUpdate:     true,
            responseUpdated:      unassignedConversation,
            expectRes: &cvconversation.WebhookMessage{
                Identity: commonidentity.Identity{
                    ID:         conversationID,
                    CustomerID: customerID,
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockReq := requesthandler.NewMockRequestHandler(mc)
            mockDB  := dbhandler.NewMockDBHandler(mc)
            h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
            ctx := context.Background()

            mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
            if tt.expectCallUpdate {
                mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, unassignFields).Return(tt.responseUpdated, nil)
            }

            res, err := h.ConversationUnassign(ctx, tt.agent, tt.conversationID)
            if tt.expectErr != nil {
                if !errors.Is(err, tt.expectErr) {
                    t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
                }
                return
            }
            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            if !reflect.DeepEqual(tt.expectRes, res) {
                t.Errorf("Wrong result.\nexpect: %v\ngot:    %v", tt.expectRes, res)
            }
        })
    }
}
```

**Step 2: Run test to verify it fails (method doesn't exist yet)**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -run Test_ConversationUnassign -v 2>&1 | head -20
```

Expected: compile error — `h.ConversationUnassign undefined`.

**Step 3: Implement ConversationUnassign in conversation.go**

Add after `ConversationUpdate`:

```go
// ConversationUnassign removes the owner from the given conversation.
// Admin/manager may unassign any conversation. The owning agent may unassign themselves.
func (h *serviceHandler) ConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ConversationUnassign",
		"customer_id":     a.CustomerID,
		"username":        a.DisplayName(),
		"conversation_id": conversationID,
	})
	log.Debug("Unassigning the conversation.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		log.Errorf("Could not get conversation info. err: %v", err)
		return nil, fmt.Errorf("%w: could not find conversation info", err)
	}

	isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	isOwningAgent := a.IsAgent() && a.Agent != nil &&
		c.OwnerType == commonidentity.OwnerTypeAgent &&
		c.OwnerID == a.Agent.ID

	if !isAdminOrManager && !isOwningAgent {
		log.Info("Caller has no permission to unassign the conversation.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, map[cvconversation.Field]any{
		cvconversation.FieldOwnerID: uuid.Nil,
	})
	if err != nil {
		log.Errorf("Could not unassign the conversation. err: %v", err)
		return nil, errors.Wrap(err, "could not unassign the conversation")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

**Step 4: Run tests**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -run Test_ConversationUnassign -v
```

Expected: all cases pass.

**Step 5: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/conversation.go bin-api-manager/pkg/servicehandler/conversation_test.go
git commit -m "NOJIRA-Conversation-unassign-design

- bin-api-manager: Add ConversationUnassign servicehandler method and tests"
```

---

## Task 6: Add ServiceAgentConversationUpdate and ServiceAgentConversationUnassign

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go`
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_conversation_test.go`

**Step 1: Write the failing tests first**

Add to `serviceagent_conversation_test.go`:

```go
func Test_ServiceAgentConversationUpdate(t *testing.T) {
	customerID     := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
	agentID        := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	conversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: customerID,
		},
	}
	updateFields := map[cvconversation.Field]any{
		cvconversation.FieldName: "updated name",
	}

	type test struct {
		name           string
		agent          *auth.AuthIdentity
		conversationID uuid.UUID
		fields         map[cvconversation.Field]any

		responseConversation *cvconversation.Conversation
		expectCallUpdate     bool
		responseUpdated      *cvconversation.Conversation

		expectErr error
		expectRes *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			name: "admin updates conversation",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID:       conversationID,
			fields:               updateFields,
			responseConversation: conversation,
			expectCallUpdate:     true,
			responseUpdated:      conversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "manager updates conversation",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			conversationID:       conversationID,
			fields:               updateFields,
			responseConversation: conversation,
			expectCallUpdate:     true,
			responseUpdated:      conversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         conversationID,
					CustomerID: customerID,
				},
			},
		},
		{
			name: "agent (non-admin) — permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			fields:               updateFields,
			responseConversation: conversation,
			expectCallUpdate:     false,
			expectErr:            serviceerrors.ErrPermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB  := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			if tt.expectCallUpdate {
				mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, tt.fields).Return(tt.responseUpdated, nil)
			}

			res, err := h.ServiceAgentConversationUpdate(ctx, tt.agent, tt.conversationID, tt.fields)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong result.\nexpect: %v\ngot:    %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentConversationUnassign(t *testing.T) {
	customerID     := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	conversationID := uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5")
	owningAgentID  := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")
	otherAgentID   := uuid.FromStringOrNil("a01f2c3a-3001-11f0-9d11-2bd5b4a45af1")

	assignedConversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   owningAgentID,
		},
	}
	unassignedConversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         conversationID,
			CustomerID: customerID,
		},
	}
	unassignFields := map[cvconversation.Field]any{
		cvconversation.FieldOwnerID: uuid.Nil,
	}

	type test struct {
		name                 string
		agent                *auth.AuthIdentity
		conversationID       uuid.UUID
		responseConversation *cvconversation.Conversation
		expectCallUpdate     bool
		responseUpdated      *cvconversation.Conversation
		expectErr            error
		expectRes            *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			name: "admin unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
		{
			name: "manager unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d6a401d4-3002-11f0-9c79-b3a64c98c8d9"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
		{
			name: "owning agent self-unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
		{
			name: "non-owning agent — permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			responseConversation: assignedConversation,
			expectCallUpdate:     false,
			expectErr:            serviceerrors.ErrPermissionDenied,
		},
		{
			name: "owning agent on already-unassigned conversation — permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         owningAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			conversationID:       conversationID,
			responseConversation: unassignedConversation,
			expectCallUpdate:     false,
			expectErr:            serviceerrors.ErrPermissionDenied,
		},
		{
			name: "admin on already-unassigned conversation — idempotent 200",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0a4c108-3002-11f0-8a3e-c33b1aef2e49"),
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			conversationID:       conversationID,
			responseConversation: unassignedConversation,
			expectCallUpdate:     true,
			responseUpdated:      unassignedConversation,
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB  := dbhandler.NewMockDBHandler(mc)
			h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			if tt.expectCallUpdate {
				mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, unassignFields).Return(tt.responseUpdated, nil)
			}

			res, err := h.ServiceAgentConversationUnassign(ctx, tt.agent, tt.conversationID)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong result.\nexpect: %v\ngot:    %v", tt.expectRes, res)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail (methods don't exist yet)**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -run "Test_ServiceAgentConversationUpdate|Test_ServiceAgentConversationUnassign" -v 2>&1 | head -10
```

Expected: compile error — methods undefined.

**Step 3: Implement both methods in serviceagent_conversation.go**

Append to the end of `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go`:

```go
// ServiceAgentConversationUpdate updates the conversation of the given id.
// Admin and manager callers only. Owning agents must use ServiceAgentConversationUnassign instead.
func (h *serviceHandler) ServiceAgentConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, fields)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not update conversation.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentConversationUnassign removes the owner from the given conversation.
// Admin/manager may unassign any conversation. The owning agent may unassign themselves.
func (h *serviceHandler) ServiceAgentConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	c, err := h.conversationGet(ctx, conversationID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get conversation.")
	}

	isAdminOrManager := h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	isOwningAgent := a.Agent != nil &&
		c.OwnerType == commonidentity.OwnerTypeAgent &&
		c.OwnerID == a.Agent.ID

	if !isAdminOrManager && !isOwningAgent {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.ConversationV1ConversationUpdate(ctx, conversationID, map[cvconversation.Field]any{
		cvconversation.FieldOwnerID: uuid.Nil,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Could not unassign conversation.")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

You will need to add these imports to `serviceagent_conversation.go` if not already present:
```go
amagent "monorepo/bin-agent-manager/models/agent"
commonidentity "monorepo/bin-common-handler/models/identity"
"github.com/gofrs/uuid"
```

**Step 4: Run tests**

```bash
cd bin-api-manager
go test ./pkg/servicehandler/... -run "Test_ServiceAgentConversation" -v
```

Expected: all cases pass.

**Step 5: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/serviceagent_conversation.go bin-api-manager/pkg/servicehandler/serviceagent_conversation_test.go
git commit -m "NOJIRA-Conversation-unassign-design

- bin-api-manager: Add ServiceAgentConversationUpdate and ServiceAgentConversationUnassign methods and tests"
```

---

## Task 7: Add server-layer handlers

**Files:**
- Modify: `bin-api-manager/server/conversations.go`
- Modify: `bin-api-manager/server/service_agents_conversations.go`

**Step 1: Add PostConversationsIdUnassign to conversations.go**

Append to the end of `bin-api-manager/server/conversations.go`:

```go
func (h *server) PostConversationsIdUnassign(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostConversationsIdUnassign",
		"request_address": c.ClientIP,
		"conversation_id": id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"auth": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ConversationUnassign(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not unassign the conversation. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
```

**Step 2: Add PutServiceAgentsConversationsId to service_agents_conversations.go**

Append to the end of `bin-api-manager/server/service_agents_conversations.go`:

```go
func (h *server) PutServiceAgentsConversationsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutServiceAgentsConversationsId",
		"request_address": c.ClientIP,
		"conversation_id": id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutServiceAgentsConversationsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	raw, err := structToFilteredMap(req)
	if err != nil {
		log.Errorf("Could not convert fields. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "Could not convert request fields.").Wrap(err))
		return
	}

	fields, err := cvconversation.ConvertStringMapToFieldMap(raw)
	if err != nil {
		log.Errorf("Could not convert fields. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "Could not convert request fields.").Wrap(err))
		return
	}

	res, err := h.serviceHandler.ServiceAgentConversationUpdate(c.Request.Context(), a, target, fields)
	if err != nil {
		log.Errorf("Could not update the conversation. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostServiceAgentsConversationsIdUnassign(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostServiceAgentsConversationsIdUnassign",
		"request_address": c.ClientIP,
		"conversation_id": id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.ServiceAgentConversationUnassign(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not unassign the conversation. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}
```

You will need to add to `service_agents_conversations.go` imports if not already present:
```go
cvconversation "monorepo/bin-conversation-manager/models/conversation"
```

**Step 3: Run full verification**

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all steps pass with zero errors.

**Step 4: Commit**

```bash
git add bin-api-manager/server/conversations.go bin-api-manager/server/service_agents_conversations.go
git commit -m "NOJIRA-Conversation-unassign-design

- bin-api-manager: Add PostConversationsIdUnassign server handler
- bin-api-manager: Add PutServiceAgentsConversationsId server handler
- bin-api-manager: Add PostServiceAgentsConversationsIdUnassign server handler"
```

---

## Task 8: Update RST docs and rebuild HTML

**Files:**
- Modify: relevant files under `bin-api-manager/docsdev/source/` (check for `conversation*.rst`)
- Modify: `bin-api-manager/docsdev/build/` (rebuild)

**Step 1: Find conversation RST docs**

```bash
ls bin-api-manager/docsdev/source/ | grep -i convers
```

**Step 2: Update relevant RST files**

For each RST file that documents conversation endpoints, add or update:
- `PUT /conversations/<id>` — note it is now **admin/manager only** (agents may no longer call it)
- `POST /conversations/<id>/unassign` — new endpoint; describe permission rules (admin/manager + owning agent), no request body, returns conversation object
- `PUT /service_agents/conversations/<id>` — new endpoint; admin/manager only; same fields as PUT /conversations/<id>
- `POST /service_agents/conversations/<id>/unassign` — same permission rules as POST /conversations/<id>/unassign

**Step 3: Clean-rebuild HTML**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: build completes with no errors.

**Step 4: Stage and commit**

```bash
cd bin-api-manager
git add docsdev/source/
git add -f docsdev/build/
git commit -m "NOJIRA-Conversation-unassign-design

- bin-api-manager: Update conversation RST docs for unassign endpoints and PUT permission change"
```

---

## Task 9: Final verification and PR

**Step 1: Run full verification across both services**

```bash
cd bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ../bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all steps pass with zero errors and zero lint warnings.

**Step 2: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: no conflicts.

**Step 3: Create PR**

```bash
gh pr create \
  --title "NOJIRA-Conversation-unassign-design" \
  --body "Add explicit unassign endpoints for conversations on both API surfaces and remove owning-agent carve-out from PUT /conversations/<id>.

- bin-openapi-manager: Add POST /conversations/{id}/unassign, PUT /service_agents/conversations/{id}, POST /service_agents/conversations/{id}/unassign paths
- bin-api-manager: Remove payloadIsExactlySelfUnassign from ConversationUpdate (PUT is now admin/manager only)
- bin-api-manager: Add ConversationUnassign (admin/manager + owning agent)
- bin-api-manager: Add ServiceAgentConversationUpdate (admin/manager only)
- bin-api-manager: Add ServiceAgentConversationUnassign (admin/manager + owning agent)
- bin-api-manager: Update conversation RST docs"
```
