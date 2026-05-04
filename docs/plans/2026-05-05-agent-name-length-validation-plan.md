# Agent Name Length Validation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Return `400 Bad Request` instead of `500` when the `name` field exceeds 255 characters on `POST /v1/agents` and `PUT /v1/agents/:id`.

**Architecture:** Add an inline length guard in the listenhandler immediately after JSON unmarshal, consistent with how UUID and JSON parse errors are already handled in the same file. No new abstractions.

**Tech Stack:** Go, gomock (go.uber.org/mock), pytest (monorepo-monitoring)

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation`

---

### Task 1: Add the constant and POST validation

**Files:**
- Modify: `bin-agent-manager/pkg/listenhandler/v1_agents.go` (lines 1–19 for constant, lines 121–125 for guard)

**Step 1: Add the constant**

After the imports block (line 18, before the first function), add:

```go
const agentNameMaxLength = 255
```

The file starts with:
```go
package listenhandler

import (
    ...
)

// agentNameMaxLength is the maximum allowed length for an agent name,
// matching the varchar(255) column in agent_agents.name.
const agentNameMaxLength = 255

// processV1AgentsGet handles GET /v1/agents request
```

**Step 2: Add the guard in processV1AgentsPost**

In `processV1AgentsPost` (around line 121), after the `json.Unmarshal` block and before the `log = log.WithFields(...)` call, add:

```go
if len(reqData.Name) > agentNameMaxLength {
    return simpleResponse(400), nil
}
```

The block should look like:

```go
var reqData request.V1DataAgentsPost
if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
    log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
    return simpleResponse(400), nil
}
if len(reqData.Name) > agentNameMaxLength {
    return simpleResponse(400), nil
}
log = log.WithFields(logrus.Fields{
```

**Step 3: Run tests to check nothing is broken yet**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation/bin-agent-manager
go test ./pkg/listenhandler/... -v -run TestProcessV1AgentsPost 2>&1 | tail -20
```

Expected: existing tests still PASS.

---

### Task 2: Add the PUT validation

**Files:**
- Modify: `bin-agent-manager/pkg/listenhandler/v1_agents.go` (lines ~346–350)

**Step 1: Add the guard in processV1AgentsIDPut**

In `processV1AgentsIDPut` (around line 346), after the `json.Unmarshal` block and before calling `h.agentHandler.UpdateBasicInfo`, add:

```go
if len(reqData.Name) > agentNameMaxLength {
    return simpleResponse(400), nil
}
```

The block should look like:

```go
var reqData request.V1DataAgentsIDPut
if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
    log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
    return simpleResponse(400), nil
}
if len(reqData.Name) > agentNameMaxLength {
    return simpleResponse(400), nil
}

tmp, err := h.agentHandler.UpdateBasicInfo(ctx, id, reqData.Name, ...)
```

**Step 2: Run existing PUT tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation/bin-agent-manager
go test ./pkg/listenhandler/... -v -run Test_processV1AgentsIDPut 2>&1 | tail -20
```

Expected: existing tests still PASS.

---

### Task 3: Add unit tests for POST name-length validation

**Files:**
- Modify: `bin-agent-manager/pkg/listenhandler/v1_agents_additional_test.go`

**Step 1: Add test cases to the existing `Test_processV1AgentsIDPut` or add a new function**

Append a new test function to `v1_agents_additional_test.go`:

```go
func Test_processV1AgentsPost_nameLengthValidation(t *testing.T) {
    tests := []struct {
        name         string
        nameField    string
        expectStatus int
    }{
        {
            name:         "name exactly 255 chars",
            nameField:    string(make([]byte, 255)), // 255 'A' equivalent; use strings.Repeat below
            expectStatus: 200,
        },
        {
            name:         "name 256 chars",
            expectStatus: 400,
        },
        {
            name:         "name 1000 chars",
            expectStatus: 400,
        },
    }
    // ...
}
```

Actually write it like the existing tests (import `strings` at top of file if not present):

```go
func Test_processV1AgentsPost_nameLengthValidation(t *testing.T) {
    tests := []struct {
        name         string
        nameLen      int
        expectStatus int
    }{
        {"255 chars - ok", 255, 200},
        {"256 chars - too long", 256, 400},
        {"1000 chars - too long", 1000, 400},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockAgent := agenthandler.NewMockAgentHandler(mc)
            h := &listenHandler{
                agentHandler: mockAgent,
            }
            ctx := context.Background()

            longName := strings.Repeat("A", tt.nameLen)
            reqBody, _ := json.Marshal(map[string]interface{}{
                "customer_id": "92883d56-7fe3-11ec-8931-37d08180a2b9",
                "username":    "testuser@example.com",
                "password":    "TestPass123",
                "name":        longName,
                "detail":      "test",
                "ring_method": "ringall",
                "permission":  16,
                "tag_ids":     []string{},
                "addresses":   []string{},
            })

            if tt.expectStatus == 200 {
                mockAgent.EXPECT().Create(
                    gomock.Any(),
                    gomock.Any(), gomock.Any(), gomock.Any(),
                    longName,
                    gomock.Any(), gomock.Any(), gomock.Any(),
                    gomock.Any(), gomock.Any(),
                ).Return(&agent.Agent{}, nil)
            }

            req := &sock.Request{
                URI:    "/v1/agents",
                Method: sock.RequestMethodPost,
                Data:   reqBody,
            }

            res, err := h.processV1AgentsPost(ctx, req)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if res.StatusCode != tt.expectStatus {
                t.Errorf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
            }
        })
    }
}
```

Also add `"strings"` to the import block at the top of `v1_agents_additional_test.go` if it is not already there.

**Step 2: Run the new test to verify it fails before the fix**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation/bin-agent-manager
go test ./pkg/listenhandler/... -v -run Test_processV1AgentsPost_nameLengthValidation 2>&1 | tail -30
```

Expected: The 256-char and 1000-char cases fail (got 200 or mock expectation mismatch).

After adding the fix in Task 1, run again — all cases should PASS.

---

### Task 4: Add unit tests for PUT name-length validation

**Files:**
- Modify: `bin-agent-manager/pkg/listenhandler/v1_agents_additional_test.go`

**Step 1: Add test cases for PUT**

Append to `v1_agents_additional_test.go`:

```go
func Test_processV1AgentsIDPut_nameLengthValidation(t *testing.T) {
    agentID := uuid.FromStringOrNil("69434cfa-79a4-11ec-a7b1-6ba5b7016d83")

    tests := []struct {
        name         string
        nameLen      int
        expectStatus int
    }{
        {"255 chars - ok", 255, 200},
        {"256 chars - too long", 256, 400},
        {"1000 chars - too long", 1000, 400},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockAgent := agenthandler.NewMockAgentHandler(mc)
            h := &listenHandler{
                agentHandler: mockAgent,
            }
            ctx := context.Background()

            longName := strings.Repeat("A", tt.nameLen)
            reqBody, _ := json.Marshal(map[string]interface{}{
                "name":        longName,
                "detail":      "test",
                "ring_method": "ringall",
            })

            if tt.expectStatus == 200 {
                mockAgent.EXPECT().UpdateBasicInfo(
                    gomock.Any(),
                    agentID,
                    longName,
                    gomock.Any(), gomock.Any(),
                ).Return(&agent.Agent{}, nil)
            }

            req := &sock.Request{
                URI:  "/v1/agents/" + agentID.String(),
                Data: reqBody,
            }

            res, err := h.processV1AgentsIDPut(ctx, req)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if res.StatusCode != tt.expectStatus {
                t.Errorf("expected status %d, got %d", tt.expectStatus, res.StatusCode)
            }
        })
    }
}
```

**Step 2: Run new test**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation/bin-agent-manager
go test ./pkg/listenhandler/... -v -run Test_processV1AgentsIDPut_nameLengthValidation 2>&1 | tail -30
```

Expected: all PASS (the fix in Task 2 is already in place).

---

### Task 5: Run full verification

**Step 1: Run full verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation/bin-agent-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps succeed with zero errors.

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation
git add bin-agent-manager/pkg/listenhandler/v1_agents.go \
        bin-agent-manager/pkg/listenhandler/v1_agents_additional_test.go
git commit -m "NOJIRA-Fix-agent-name-validation

- bin-agent-manager: Add name length validation (max 255) in POST and PUT agent handlers"
```

---

### Task 6: Create PR for monorepo

**Step 1: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-agent-name-validation
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: no conflicts.

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Fix-agent-name-validation

gh pr create \
  --title "NOJIRA-Fix-agent-name-validation" \
  --body "$(cat <<'EOF'
Return 400 Bad Request when agent name exceeds 255 characters (varchar(255) DB column limit) instead of propagating a DB error as 500.

Fixes: voipbin/monorepo-monitoring#108

- bin-agent-manager: Add agentNameMaxLength = 255 constant in listenhandler
- bin-agent-manager: Validate name length in processV1AgentsPost before calling agentHandler.Create
- bin-agent-manager: Validate name length in processV1AgentsIDPut before calling agentHandler.UpdateBasicInfo
- bin-agent-manager: Add unit tests for both handlers covering 255 (ok), 256 (400), and 1000 (400) char names
EOF
)"
```

---

### Task 7: Fix the api-validator test (monorepo-monitoring)

> **Do this AFTER the monorepo PR is merged and deployed to production.**

**Files:**
- Worktree: create in `~/gitvoipbin/monorepo-monitoring/.worktrees/NOJIRA-Fix-agent-name-validation` (or equivalent)
- Modify: `api-validator/tests/scenarios/test_agent_edge_cases.py`

**Step 1: Create worktree in monorepo-monitoring**

```bash
cd ~/gitvoipbin/monorepo-monitoring
git fetch origin main
git worktree add .worktrees/NOJIRA-Fix-agent-name-validation -b NOJIRA-Fix-agent-name-validation
cd .worktrees/NOJIRA-Fix-agent-name-validation
```

**Step 2: Update the test**

In `api-validator/tests/scenarios/test_agent_edge_cases.py`, replace `test_create_agent_with_very_long_name`:

```python
@pytest.mark.agents
@pytest.mark.edge_cases
def test_create_agent_with_very_long_name(api_client, cleanup_agents):
    """Test that names within 255 chars are accepted and names >255 chars return 400."""
    # Names within the DB column limit should succeed
    for length in [50, 100, 255]:
        agent_data = AgentFactory.build(name="A" * length)
        response = api_client.post("/agents", json=agent_data)
        assert response.status_code == 200, \
            f"Expected 200 for name length {length}, got {response.status_code}: {response.text}"
        agent = AgentResponse(**response.json())
        cleanup_agents.append(agent.id)

    # Names exceeding 255 chars must be rejected with 400
    agent_data = AgentFactory.build(name="A" * 1000)
    response = api_client.post("/agents", json=agent_data)
    assert response.status_code == 400, \
        f"Expected 400 for name length 1000, got {response.status_code}: {response.text}"
```

**Step 3: Run the test against production**

```bash
cd ~/gitvoipbin/monorepo-monitoring/.worktrees/NOJIRA-Fix-agent-name-validation/api-validator
pytest tests/scenarios/test_agent_edge_cases.py::test_create_agent_with_very_long_name -v
```

Expected: PASS.

**Step 4: Commit and create PR**

```bash
git add api-validator/tests/scenarios/test_agent_edge_cases.py
git commit -m "NOJIRA-Fix-agent-name-validation

- api-validator: Tighten test_create_agent_with_very_long_name to assert 200 for ≤255 chars and 400 for >255 chars"

git push -u origin NOJIRA-Fix-agent-name-validation

gh pr create \
  --title "NOJIRA-Fix-agent-name-validation" \
  --body "$(cat <<'EOF'
Tighten test_create_agent_with_very_long_name to reflect the server-side validation added in voipbin/monorepo#<PR>.

Previously the test accepted 200 or 400 for any name length. Now:
- 50, 100, 255 chars → assert 200
- 1000 chars → assert 400

- api-validator: Update test_create_agent_with_very_long_name assertions
EOF
)"
```
