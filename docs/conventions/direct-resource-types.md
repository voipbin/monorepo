# No Magic Strings for Direct Resource Types

### 16.1 Policy

All direct resource type strings (`"ai"`, `"ai_team"`, `"agent"`, `"queue"`, `"conference"`, `"extension"`) MUST use the constants defined in `bin-direct-manager/models/direct/direct.go`:

```go
import dmdirect "monorepo/bin-direct-manager/models/direct"

// CORRECT — use constants
ResourceType: dmdirect.ResourceTypeAI,
ResourceType: dmdirect.ResourceTypeAITeam,

// WRONG — hardcoded strings
ResourceType: "ai",
ResourceType: "ai_team",
```

**Available constants:**

| Constant | Value |
|---|---|
| `dmdirect.ResourceTypeAI` | `"ai"` |
| `dmdirect.ResourceTypeAITeam` | `"ai_team"` |
| `dmdirect.ResourceTypeAgent` | `"agent"` |
| `dmdirect.ResourceTypeQueue` | `"queue"` |
| `dmdirect.ResourceTypeConference` | `"conference"` |
| `dmdirect.ResourceTypeExtension` | `"extension"` |

**Exception:** The constant definition file itself (`bin-direct-manager/models/direct/direct.go`) and test data in `bin-direct-manager/pkg/dbhandler/` may use string literals. Add `// nolint:magicstring` to suppress false positives.

### 16.2 Pre-commit Hook

A pre-commit hook at `.githooks/pre-commit` enforces this policy by scanning staged `.go` files for hardcoded resource type strings.

**Installation:**

```bash
git config core.hooksPath .githooks
```

The hook flags lines matching `ResourceType: "ai"` or similar patterns. To suppress a false positive, add `// nolint:magicstring` on the same line.

### 16.3 Cross-Service Normalization

When a resource type from the `direct` domain needs to map to a different type in another service, the translation belongs in `bin-api-manager` — not in the downstream service.

Example: `dmdirect.ResourceTypeAITeam` (`"ai_team"`) maps to `amaicall.AssistanceTypeTeam` (`"team"`) in the ai-manager. The api-manager normalizes before calling ai-manager:

```go
// bin-api-manager/pkg/servicehandler/aicall.go
if assistanceType == amaicall.AssistanceType(dmdirect.ResourceTypeAITeam) {
    assistanceType = amaicall.AssistanceTypeTeam
}
```

Downstream services (ai-manager, call-manager, etc.) should NOT know about `"ai_team"` — they only work with their own domain types.
