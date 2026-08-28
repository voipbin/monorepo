# Architecture: bin-agent-manager

## Component Overview

```mermaid
graph TD
    CMD["cmd/agent-manager"] --> LH["pkg/listenhandler\n(RabbitMQ RPC server)"]
    CMD --> SH["pkg/subscribehandler\n(Event consumer)"]

    LH --> AH["pkg/agenthandler\n(Agent business logic)"]
    LH --> MH["pkg/metricshandler\n(Prometheus metrics)"]

    AH --> DBH["pkg/dbhandler\n(MySQL via Squirrel)"]
    AH --> Cache["pkg/cachehandler\n(Redis)"]
```

## Layer Responsibilities

| Package | Role | Key Types |
|---------|------|-----------|
| `pkg/agenthandler` | Core agent logic: CRUD, login/auth, status management, permission updates, password reset, address management | `agent.Agent`, `agent.Status`, `agent.Permission` |
| `pkg/metricshandler` | Prometheus metrics registration and recording for agent operations | Prometheus counter/histogram types |
| `pkg/listenhandler` | RabbitMQ RPC request router (regex pattern matching) | `sock.Request`, `sock.Response` |
| `pkg/subscribehandler` | Consumes events from call-manager, customer-manager, and webhook-manager to react to external state changes | queue event structs |
| `pkg/dbhandler` | MySQL CRUD using Squirrel query builder | all model structs |
| `pkg/cachehandler` | Redis fast-path lookups for agents | `agent.Agent` |
| `models/agent` | Agent data model, status constants, permission flags, ring method | `agent.Agent`, `agent.Status`, `agent.Permission` |

## Request Routing

Requests arrive via RabbitMQ queue `bin-manager.agent-manager.request`. The `listenhandler` matches each request's URI against regex patterns and dispatches to the appropriate handler function.

| Route Pattern | Method | Description |
|---------------|--------|-------------|
| `/v1/agents/count_by_customer$` | GET | Count agents by customer ID |
| `/v1/agents$` | POST | Create a new agent |
| `/v1/agents\?(.*)`| GET | List agents with filters/pagination |
| `/v1/agents/{{UUID}}/login$` | POST | Authenticate an agent (returns token) |
| `/v1/agents/{{UUID}}$` | GET/PUT/DELETE | Get, update, or delete an agent |
| `/v1/agents/{{UUID}}/addresses$` | GET/POST/DELETE | Manage agent SIP/contact addresses |
| `/v1/agents/{{UUID}}/tag_ids$` | PUT | Update agent tag IDs |
| `/v1/agents/{{UUID}}/status$` | PUT | Update agent status (available/away/busy/offline) |
| `/v1/agents/{{UUID}}/password$` | PUT | Change agent password |
| `/v1/agents/{{UUID}}/permission$` | PUT | Update agent permission flags |
| `/v1/agents/{{UUID}}/direct-hash-regenerate$` | POST | Regenerate the direct-access hash |
| `/v1/agents/get_by_customer_id_address$` | GET | Look up an agent by customer ID and SIP address |
| `/v1/login$` | POST | Global login endpoint (all agents) |
| `/v1/password-forgot$` | POST | Initiate password reset flow (send email) |
| `/v1/password-reset$` | POST | Complete password reset with token |

## Events Published

Exchange: `bin-manager.agent-manager.event` (fanout, system of record) and — since VOIP-1405 — the global topic exchange `bin-manager.event`.

Both `cmd/agent-manager` and `cmd/agent-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange (unchanged) and once to the global topic exchange with the routing key `agent-manager.<resource>.<subscription-id>.<action>`. Both construction sites must stay in lockstep on this option — enabling it in only one would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish.

Every agent event is addressed by the agent's OWN id (the default top-level `id` fallback; no `eventtopic.SubscriptionIdentifier` override exists in this service), so a consumer following one agent binds `agent-manager.agent.<agent-id>.#` and receives all four event types. Note `agent_status_updated` splits on the FIRST underscore, keeping the resource segment `agent` and putting `status_updated` in the action segment. The exact keys are pinned by `models/agent/routingkey_golden_test.go`; the schema lives in the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md`.

| Event type | Trigger | Publish path |
|-----------|---------|--------------|
| `agent.EventTypeAgentCreated` | Agent created | `PublishWebhookEvent` |
| `agent.EventTypeAgentUpdated` | Agent info, addresses, tag IDs, password, or permission updated | `PublishEvent` + `PublishWebhookEvent` |
| `agent.EventTypeAgentDeleted` | Agent deleted | `PublishWebhookEvent` |
| `agent.EventTypeAgentStatusUpdated` | Agent status changed (available/away/busy/offline/ringing) | `PublishWebhookEvent` |
