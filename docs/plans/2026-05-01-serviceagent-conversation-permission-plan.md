# ServiceAgent Conversation Permission Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Admin and manager callers at `GET /service_agents/conversations` and `GET /service_agents/conversations/{id}` should see all conversations belonging to their customer, not just those they personally own.

**Architecture:** Add a `hasPermission` guard in `ServiceAgentConversationList` and `ServiceAgentConversationGet` inside `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go`. Admin/manager callers get a customer-scoped filter (or customer-level ownership check); regular agents keep the current owner-scoped behaviour. No OpenAPI or RST changes required.

**Tech Stack:** Go, gomock (`go.uber.org/mock/gomock`), `bin-agent-manager/models/agent` permission constants.

---

### Task 1: Add failing tests for admin/manager in `ServiceAgentConversationList`

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_conversation_test.go`

**Step 1: Add two new test cases to `Test_ServiceAgentConversationList`**

Insert after the existing `"normal"` case (after line 151, before the closing `}`):

```go
{
    name: "admin sees all conversations (no owner filter)",

    agent: auth.NewAgentIdentity(&amagent.Agent{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Permission: amagent.PermissionCustomerAdmin,
    }),
    size:  100,
    token: "2021-03-01T01:00:00.995000Z",

    responseConversations: []cvconversation.Conversation{
        {
            Identity: commonidentity.Identity{
                ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
            },
        },
    },

    expectFilters: map[cvconversation.Field]any{
        cvconversation.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        cvconversation.FieldDeleted:    false,
    },
    expectRes: []*cvconversation.WebhookMessage{
        {
            Identity: commonidentity.Identity{
                ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
            },
        },
    },
},
{
    name: "manager sees all conversations (no owner filter)",

    agent: auth.NewAgentIdentity(&amagent.Agent{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09b"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Permission: amagent.PermissionCustomerManager,
    }),
    size:  100,
    token: "2021-03-01T01:00:00.995000Z",

    responseConversations: []cvconversation.Conversation{
        {
            Identity: commonidentity.Identity{
                ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
            },
        },
    },

    expectFilters: map[cvconversation.Field]any{
        cvconversation.FieldCustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        cvconversation.FieldDeleted:    false,
    },
    expectRes: []*cvconversation.WebhookMessage{
        {
            Identity: commonidentity.Identity{
                ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
            },
        },
    },
},
```

**Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-ServiceAgent-conversation-permission/bin-api-manager
go test ./pkg/servicehandler/... -run Test_ServiceAgentConversationList -v 2>&1 | tail -20
```

Expected: FAIL — the admin/manager cases will call `ConversationV1ConversationList` with the wrong filter (OwnerID instead of CustomerID).

---

### Task 2: Add failing tests for admin/manager in `ServiceAgentConversationGet`

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_conversation_test.go`

**Step 1: Add new test cases to `Test_ServiceAgentConversationGet`**

Insert after the existing `"normal"` case (after line 63, before the closing `}`):

```go
{
    name: "admin gets conversation owned by another agent",

    agent: auth.NewAgentIdentity(&amagent.Agent{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("aaaaaaaa-3b9f-11ef-98ac-db226570f09a"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Permission: amagent.PermissionCustomerAdmin,
    }),
    conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

    responseConversation: &cvconversation.Conversation{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Owner: commonidentity.Owner{
            OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"), // different agent
        },
    },
    expectRes: &cvconversation.WebhookMessage{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Owner: commonidentity.Owner{
            OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
        },
    },
},
{
    name: "manager gets conversation owned by another agent",

    agent: auth.NewAgentIdentity(&amagent.Agent{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("bbbbbbbb-3b9f-11ef-98ac-db226570f09b"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Permission: amagent.PermissionCustomerManager,
    }),
    conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

    responseConversation: &cvconversation.Conversation{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Owner: commonidentity.Owner{
            OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"), // different agent
        },
    },
    expectRes: &cvconversation.WebhookMessage{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
            CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
        },
        Owner: commonidentity.Owner{
            OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
        },
    },
},
```

**Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-ServiceAgent-conversation-permission/bin-api-manager
go test ./pkg/servicehandler/... -run Test_ServiceAgentConversationGet -v 2>&1 | tail -20
```

Expected: FAIL — admin/manager cases will be blocked by the `OwnerID != AgentID` check.

---

### Task 3: Implement the fix in `ServiceAgentConversationList`

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go`

**Step 1: Replace the `ServiceAgentConversationList` body**

Current code (lines 40-68):
```go
func (h *serviceHandler) ServiceAgentConversationList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cvconversation.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[cvconversation.Field]any{
		cvconversation.FieldDeleted: false,
		cvconversation.FieldOwnerID: a.AgentID(),
	}

	tmps, err := h.conversationList(ctx, a, size, token, filters)
	...
```

Replace the filter block with:
```go
	// Admin and manager callers see all conversations for their customer.
	// Regular agents see only conversations they own.
	filters := map[cvconversation.Field]any{
		cvconversation.FieldDeleted: false,
	}
	if h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		filters[cvconversation.FieldCustomerID] = a.CustomerID
	} else {
		filters[cvconversation.FieldOwnerID] = a.AgentID()
	}
```

Also add the missing import at the top if not already present:
```go
amagent "monorepo/bin-agent-manager/models/agent"
```

**Step 2: Run the list tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-ServiceAgent-conversation-permission/bin-api-manager
go test ./pkg/servicehandler/... -run Test_ServiceAgentConversationList -v 2>&1 | tail -20
```

Expected: all cases PASS.

---

### Task 4: Implement the fix in `ServiceAgentConversationGet`

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/serviceagent_conversation.go`

**Step 1: Replace the ownership check in `ServiceAgentConversationGet`**

Current code:
```go
	if tmp.OwnerID != a.AgentID() {
		return nil, serviceerrors.ErrPermissionDenied
	}
```

Replace with:
```go
	isAdminOrManager := h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager)
	if !isAdminOrManager && tmp.OwnerID != a.AgentID() {
		return nil, serviceerrors.ErrPermissionDenied
	}
```

**Step 2: Run the get tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-ServiceAgent-conversation-permission/bin-api-manager
go test ./pkg/servicehandler/... -run Test_ServiceAgentConversationGet -v 2>&1 | tail -20
```

Expected: all cases PASS.

---

### Task 5: Run full verification workflow

**Step 1: Run all verification steps**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-ServiceAgent-conversation-permission/bin-api-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps succeed with no errors.

---

### Task 6: Commit, push, and create PR

**Step 1: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-ServiceAgent-conversation-permission
git add bin-api-manager/pkg/servicehandler/serviceagent_conversation.go \
        bin-api-manager/pkg/servicehandler/serviceagent_conversation_test.go \
        docs/plans/
git commit -m "NOJIRA-ServiceAgent-conversation-permission

- bin-api-manager: admin/manager callers at GET /service_agents/conversations now see all customer conversations (filter by CustomerID)
- bin-api-manager: admin/manager callers at GET /service_agents/conversations/{id} can fetch any conversation belonging to their customer
- bin-api-manager: add test cases for admin and manager permission in ServiceAgentConversationList and ServiceAgentConversationGet"
```

**Step 2: Push**

```bash
git push -u origin NOJIRA-ServiceAgent-conversation-permission
```

**Step 3: Create PR**

```bash
gh pr create \
  --title "NOJIRA-ServiceAgent-conversation-permission" \
  --body "Admin and manager agents at GET /service_agents/conversations now see all conversations for their customer instead of only their own.

- bin-api-manager: ServiceAgentConversationList uses CustomerID filter for admin/manager, OwnerID filter for regular agents
- bin-api-manager: ServiceAgentConversationGet allows admin/manager to fetch any conversation in their customer scope
- bin-api-manager: new test cases cover admin and manager paths in both functions"
```
