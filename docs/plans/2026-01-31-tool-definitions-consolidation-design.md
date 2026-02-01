# Tool Definitions Consolidation Design

**Date:** 2026-01-31
**Status:** Proposed
**Problem:** Tool definitions are split between pipecat-manager and ai-manager, causing duplication, maintenance issues, and inflexibility

## Problem Statement

Currently, LLM tool functions are defined in pipecat-manager (Python) while the business logic lives in ai-manager (Go). This causes:

1. **Wrong service ownership** - Tool definitions should be in ai-manager since it owns the business logic
2. **Inflexibility** - Adding/modifying tools requires changes in multiple services
3. **Duplication** - Tool names and structures defined in multiple places

## Solution Overview

1. **ai-manager becomes single source of truth** for tool definitions
2. **AI model gains a `ToolNames` field** for per-AI tool configuration
3. **New API endpoint** `GET /v1/tools` returns all tool definitions
4. **pipecat-manager fetches at startup** and caches tool catalog
5. **Per-call filtering** based on AI's enabled tools

## Architecture

**Current State:**
```
pipecat-manager (Python)     pipecat-manager (Go)      ai-manager
├── Tool definitions ───────► Proxy ──────────────────► Tool execution
├── Tool registration        (forwards requests)       (business logic)
└── LLM integration
```

**Proposed State:**
```
ai-manager                    pipecat-manager (Go)      pipecat-manager (Python)
├── Tool definitions ◄─────── Startup fetch ◄───────── Cache & register
├── Tool execution            Proxy                     LLM integration
└── Per-AI tool config
```

## Data Model Changes

### Tool Name Type

New file: `bin-ai-manager/models/tool/main.go`

```go
package tool

type ToolName string

const (
    ToolNameAll               ToolName = "all"
    ToolNameConnectCall       ToolName = "connect_call"
    ToolNameGetVariables      ToolName = "get_variables"
    ToolNameGetAIcallMessages ToolName = "get_aicall_messages"
    ToolNameSendEmail         ToolName = "send_email"
    ToolNameSendMessage       ToolName = "send_message"
    ToolNameSetVariables      ToolName = "set_variables"
    ToolNameStopFlow          ToolName = "stop_flow"
    ToolNameStopMedia         ToolName = "stop_media"
    ToolNameStopService       ToolName = "stop_service"
)

type Tool struct {
    Name        ToolName       `json:"name"`
    Description string         `json:"description"`
    Parameters  map[string]any `json:"parameters"`
}
```

### AI Model Update

Modify: `bin-ai-manager/models/ai/main.go`

```go
type AI struct {
    // ... existing fields ...

    // Enabled tools for this AI
    // [ToolNameAll] = all tools, specific names = those tools only, [] or NULL = none
    ToolNames []ToolName `json:"tool_names,omitempty" db:"tool_names,json"`

    // ...
}
```

### Tool Configuration Format

- `["all"]` - All tools enabled
- `["connect_call", "send_email"]` - Specific tools only
- `[]` or `NULL` - No tools enabled (AI can only talk)

## API Endpoint

### GET /v1/tools

Returns all tool definitions. Pipecat calls this once at startup.

**Response:**
```json
{
    "tools": [
        {
            "name": "connect_call",
            "description": "Connects to another endpoint...\n\nWHEN TO USE:\n...",
            "parameters": {
                "type": "object",
                "properties": {
                    "run_llm": { "type": "boolean", "description": "..." },
                    "destinations": { ... }
                },
                "required": ["destinations"]
            }
        }
    ]
}
```

## Pipecat Integration

### Go Side

At startup, fetch and cache tool definitions:

```go
// pkg/pipecatcallhandler/main.go
type pipecatcallHandler struct {
    // ... existing fields ...
    toolCache []tool.Tool  // Cached tool definitions
}

func (h *pipecatcallHandler) InitToolCache(ctx context.Context) error {
    tools, err := h.requestHandler.AIV1ToolsGet(ctx)
    if err != nil {
        return err
    }
    h.toolCache = tools
    return nil
}
```

When starting a pipecatcall, filter and pass enabled tools to Python:

```go
func (h *pipecatcallHandler) getEnabledTools(ai *ai.AI) []tool.Tool {
    if len(ai.ToolNames) == 0 {
        return nil // No tools
    }
    if ai.ToolNames[0] == tool.ToolNameAll {
        return h.toolCache // All tools
    }
    // Filter to only enabled tools
    var enabled []tool.Tool
    for _, t := range h.toolCache {
        for _, name := range ai.ToolNames {
            if t.Name == name {
                enabled = append(enabled, t)
                break
            }
        }
    }
    return enabled
}
```

### Python Side

Receive tools from Go instead of defining them:

```python
# run.py - receive tools in /run request
@app.post("/run")
async def run(request: RunRequest):
    tools = request.tools  # List of tool definitions from Go
    tool_register(llm_service, pipecatcall_id, tools)
```

```python
# tools.py - simplified, just registration and execution
def tool_register(llm_service, pipecatcall_id, tools):
    for tool in tools:
        wrapper = create_wrapper(tool["name"], pipecatcall_id)
        llm_service.register_function(tool["name"], wrapper)
```

## Database Migration

```sql
-- Add tool_names column to ai table
ALTER TABLE ai ADD COLUMN tool_names JSON DEFAULT NULL;

-- Backwards compatibility: existing AIs get all tools enabled
UPDATE ai SET tool_names = '["all"]' WHERE tm_delete = '';
```

Note: Both `NULL` and `[]` mean no tools enabled.

## File Changes

### Files to Create

| File | Description |
|------|-------------|
| `bin-ai-manager/models/tool/main.go` | ToolName type, Tool struct |
| `bin-ai-manager/pkg/toolhandler/main.go` | Tool handler interface |
| `bin-ai-manager/pkg/toolhandler/definitions.go` | All 9 tool definitions |
| `bin-dbscheme-manager/alembic/versions/xxx_add_tool_names_to_ai.py` | Migration |

### Files to Modify

| File | Changes |
|------|---------|
| `bin-ai-manager/models/ai/main.go` | Add ToolNames field |
| `bin-ai-manager/pkg/listenhandler/` | Add GET /v1/tools endpoint |
| `bin-pipecat-manager/pkg/pipecatcallhandler/` | Add tool cache, fetch at startup |
| `bin-pipecat-manager/scripts/pipecat/run.py` | Receive tools from request |
| `bin-pipecat-manager/scripts/pipecat/tools.py` | Remove definitions, keep execution logic |

### Files to Eventually Remove

- Tool definitions in `tools.py` (after migration complete)

## Deployment Order

1. Deploy ai-manager with new endpoint and model
2. Run migration to set `["all"]` for existing AIs
3. Deploy pipecat-manager to fetch tools from ai-manager
4. Remove old tool definitions from pipecat Python code

## Rollback Safety

- If pipecat can't reach ai-manager at startup, fail fast with clear error
- Old pipecat-manager still works until Python tool definitions removed
