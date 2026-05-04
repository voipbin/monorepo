# Design: Agent Name Length Validation

**Date:** 2026-05-05
**Issue:** voipbin/monorepo-monitoring#108
**Branch:** NOJIRA-Fix-agent-name-validation

## Problem

`POST /v1/agents` and `PUT /v1/agents/:id` pass the `name` field to MySQL without length
validation. The `agent_agents.name` column is `varchar(255)`. When a name longer than 255
characters reaches the DB layer, MySQL returns an error that propagates as an unhandled
`500 Internal Server Error` instead of a clean `400 Bad Request`.

The api-validator test `test_create_agent_with_very_long_name` asserts
`status_code in [200, 400]` but receives `500`, so the test fails.

## Root Cause

`processV1AgentsPost` and `processV1AgentsIDPut` in
`bin-agent-manager/pkg/listenhandler/v1_agents.go` unmarshal the request and call the
business layer immediately, with no field-length guard.

## Solution (Approach A)

Add inline length validation in the listenhandler, consistent with how other input errors
(invalid UUID, JSON parse failure) are already handled in the same file.

### Changes

#### 1. `bin-agent-manager/pkg/listenhandler/v1_agents.go`

Add a package-level constant:

```go
const agentNameMaxLength = 255
```

In `processV1AgentsPost`, after `json.Unmarshal`:

```go
if len(reqData.Name) > agentNameMaxLength {
    return simpleResponse(400), nil
}
```

Apply the same guard in `processV1AgentsIDPut`.

#### 2. `bin-agent-manager/pkg/listenhandler/` — unit tests

Add table-driven test cases covering:
- Name exactly 255 chars → 200
- Name 256 chars → 400
- Name 1000 chars → 400

#### 3. `monorepo-monitoring/api-validator/tests/scenarios/test_agent_edge_cases.py`

Tighten `test_create_agent_with_very_long_name`:
- 50, 100, 255 chars → assert exactly 200
- 1000 chars → assert exactly 400

### Non-changes

- No DB migration needed (column limit unchanged).
- No change to agenthandler or dbhandler.
- No new error types or abstractions.

## Verification

```bash
# monorepo
cd bin-agent-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# api-validator (after deploying fix)
pytest tests/scenarios/test_agent_edge_cases.py::test_create_agent_with_very_long_name -v
```
