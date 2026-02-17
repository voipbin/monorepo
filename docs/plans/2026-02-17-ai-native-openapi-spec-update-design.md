# AI-Native OpenAPI Spec Update Design

## Problem Statement

The VoIPbin OpenAPI specification (`bin-openapi-manager/openapi/openapi.yaml`, ~4518 lines, ~40 schemas) lacks the metadata that AI agents need to correctly understand and use the API. UUID fields have no `format:`, timestamps are untyped strings, descriptions are vague, examples are missing, and polymorphic fields use `additionalProperties: true` instead of explicit schemas.

This causes AI agents to hallucinate field formats, guess foreign-key relationships, and send malformed requests.

## AI-Native Rules Being Applied

Five rules from `bin-openapi-manager/CLAUDE.md`:

1. **Use `oneOf` for Polymorphism** - Replace `additionalProperties: true` with explicit `oneOf`
2. **Strict Structured Strings** - Add `format:`, `pattern:`, or `example:` to all string fields
3. **Provenance in Descriptions** - ID reference fields must state where the ID comes from
4. **Mandatory Realistic Examples** - Every leaf property needs a real-looking `example:`
5. **Explicit Array Constraints** - Arrays needing at least one item get `minItems: 1`

## Key Design Decisions

### Decision 1: `x-go-type: string` bridge strategy (Rule 2)

**Problem:** Adding `format: uuid` changes the generated Go type from `*string` to `*openapi_types.UUID`. Adding `format: date-time` changes `*string` to `*time.Time`. This would break all 33 dependent services.

**Solution:** oapi-codegen v2.5.1 supports the `x-go-type` extension. When present, it takes precedence over `format` (checked first at line 289 of `schema.go`, triggers early return). This lets us add format specifiers for AI readability while keeping Go types unchanged:

```yaml
id:
  type: string
  format: uuid
  x-go-type: string          # Go stays *string, AI sees uuid
  example: "550e8400-e29b-41d4-a716-446655440000"
```

**Trade-off:** `x-go-type: string` is a bridge. When ready to adopt `*openapi_types.UUID` across all services, remove the `x-go-type` overrides in a follow-up PR.

### Decision 2: No phone number `pattern:` on polymorphic fields

**Problem:** `CommonAddress.target` holds UUIDs, emails, SIP URIs, or phone numbers depending on the `type` field. Adding `pattern: "^\\+[1-9]\\d{1,14}$"` would reject non-tel addresses.

`NumberManagerNumber.number` must accept both E.164 (`+14155551234`) and virtual numbers (`+899XXXXXXXXX`).

**Solution:** Use `description:` + `example:` to document formats. No `pattern:` on polymorphic fields.

### Decision 3: OpenAPI 3.0 `$ref` sibling handling

**Problem:** The spec uses `openapi: 3.0.0`. In 3.0, sibling keywords next to `$ref` are ignored by spec-compliant tools. Many enum fields use `$ref`:

```yaml
status:
  $ref: '#/components/schemas/CallManagerCallStatus'
  # description/example here would be ignored by tools
```

**Solution:** Two-pronged approach:
- Add `example:` to the **enum schema definitions** themselves (always rendered by all tools)
- Add `description:`/`example:` alongside `$ref` for AI readability (AI reads raw YAML regardless of OpenAPI spec rules; oapi-codegen silently ignores them)

Note: the existing spec already has description alongside `$ref` in several places (e.g., `CallManagerRecording.reference_type`), so this pattern is established.

### Decision 4: Keep `additionalProperties: true` for generic objects

Three schemas use `additionalProperties: true` without typed sub-schemas:
- `AIManagerAI.engine_data` — no typed schemas per engine type
- `ConferenceManagerConference.data` — custom conference data, no schema
- `TimelineManagerEvent.data` — event payload, no schema

These stay as-is. Only `FlowManagerAction.option` has 30+ typed sub-schemas to reference.

### Decision 5: Phase oneOf separately

Converting `FlowManagerAction.option` to `oneOf` changes the generated Go type from `*map[string]interface{}` to a union wrapper struct. This requires:
- Rewriting `ParseOption()` / `ConvertOption()` in flow-manager
- Updating 30+ `actionExecute*()` handlers in call-manager
- Updating conversion code in api-manager, ai-manager, campaign-manager, conference-manager, queue-manager

Cannot use `discriminator` because the `type` field is on the parent `FlowManagerAction`, not inside the option child schemas (OpenAPI 3.0 requires the discriminator property inside each child schema).

This is Phase 2 in a separate PR.

## Implementation Phases

### Phase 1: Safe Metadata (Rules 2-5, single PR, zero breaking changes)

**File changed:** Only `bin-openapi-manager/openapi/openapi.yaml`

**Changes by rule:**

| Rule | Change | Count |
|------|--------|-------|
| Rule 2 | `format: uuid` + `x-go-type: string` on all UUID fields | ~100 fields |
| Rule 2 | `format: date-time` + `x-go-type: string` on all timestamp fields | ~80 fields |
| Rule 3 | Provenance descriptions on all reference ID fields | ~50 fields |
| Rule 4 | `example:` on all plain leaf properties | ~200 fields |
| Rule 4 | `example:` on all enum schema definitions | ~30 enums |
| Rule 4 | `example:` alongside `$ref` fields for AI readability | ~60 fields |
| Rule 5 | `minItems: 1` on required arrays | ~15 arrays |

**Schema update order (top-to-bottom in openapi.yaml):**

1. Components/Parameters (PageSize, PageToken)
2. Auth (AuthLoginResponse)
3. AgentManager (AgentPermission, AgentRingMethod, AgentStatus, Agent)
4. BillingManager (AccountPaymentMethod, AccountPaymentType, AccountPlanType, Account, BillingreferenceType, BillingStatus, BillingTransactionType, Billing)
5. CallManager (CallDirection, CallHangupBy, CallHangupReason, CallMuteDirection, CallStatus, CallType, Call, GroupcallAnswerMethod, GroupcallRingMethod, GroupcallStatus, Groupcall, RecordingFormat, RecordingReferenceType, RecordingStatus, Recording)
6. CampaignManager (CampaignEndHandle, CampaignExecute, CampaignStatus, CampaignType, Campaign, CampaigncallReferenceType, CampaigncallResult, CampaigncallStatus, Campaigncall, Outplan)
7. AIManager (AIEngineType, AIEngineModel, AI, AIcallGender, AIcallReferenceType, AIcallStatus, AIcall, MessageDirection, MessageRole, Message, SummaryReferenceType, SummaryStatus, Summary)
8. Common (CommonAddress, CommonPagination)
9. ConferenceManager (ConferenceStatus, ConferenceType, Conference, ConferencecallReferenceType, ConferencecallStatus, Conferencecall)
10. ContactManager (fill remaining gaps — already partially compliant)
11. ConversationManager (AccountType, Account, ConversationReferenceType, Conversation, MediaType, Media, MessageDirection, MessageReferenceType, MessageStatus, Message)
12. CustomerManager (Accesskey, CustomerWebhookMethod, CustomerStatus, Customer)
13. EmailManager (EmailAttachmentReferenceType, EmailStatus, EmailAttachment, Email)
14. FlowManager (ActionType, all ActionOption* schemas, Action — except option field, ActiveflowStatus, ReferenceType, Activeflow, FlowType, Flow)
15. MessageManager (MessageDirection, MessageProviderName, MessageType, Message, TargetStatus, Target)
16. NumberManager (AvailableNumber, AvailableNumberFeature, NumberType, NumberProviderName, NumberStatus, Number)
17. OutdialManager (Outdial, OutdialtargetStatus, Outdialtarget)
18. QueueManager (QueueRoutingMethod, Queue, QueuecallReferenceType, QueuecallStatus, Queuecall)
19. RegistrarManager (AuthType, Extension, Trunk)
20. RouteManager (ProviderType, Provider, Route)
21. StorageManager (Account, FileReferenceType, File)
22. TalkManager (TalkType, ParticipantInput, Talk, MessageType, Message, MediaType, Media, Metadata, Reaction, Participant)
23. TimelineManager (Event)
24. TagManager (Tag)
25. TtsManager (Speaking)
26. TranscribeManager (TranscribeDirection, TranscribeReferenceType, TranscribeStatus, Transcribe, TranscriptDirection, Transcript)
27. TransferManager (TransferType, Transfer)
28. RagManager (DocType, Source, QueryResponse)

**Verification after Phase 1:**
```bash
# In bin-openapi-manager
cd bin-openapi-manager
go generate ./...
# Diff gen.go — must show ZERO type changes (only description/example in comments if any)
go test ./...
golangci-lint run -v --timeout 5m

# In bin-api-manager
cd bin-api-manager
go generate ./...
go test ./...
golangci-lint run -v --timeout 5m
```

### Phase 2: oneOf Conversion (Rule 1, separate PR, breaking changes)

**Scope:** Convert `FlowManagerAction.option` from `additionalProperties: true` to `oneOf` listing all 30+ ActionOption schemas without discriminator.

**Generated Go type change:** `*map[string]interface{}` → union wrapper struct with `AsFlowManagerActionOptionTalk()`, `FromFlowManagerActionOptionPlay()`, etc. and internal `json.RawMessage`.

**Services requiring code changes:**
- bin-flow-manager: Rewrite `ParseOption()` / `ConvertOption()`, update `Action` struct
- bin-call-manager: Update 30+ `actionExecute*()` handlers
- bin-api-manager: Update `ConvertFlowManagerAction()` conversion
- bin-ai-manager: Update tool handling
- bin-campaign-manager: Update campaign action handling
- bin-conference-manager: Update conference call actions
- bin-queue-manager: Update queue call execution

**Not in scope for this design — Phase 2 will have its own design doc.**

## What Won't Change

- Schema names/structure (no renames, no new schemas in Phase 1)
- Path YAML files (only `openapi.yaml` schemas section)
- Existing enum values / `x-enum-varnames`
- Go generated types (verified by diffing `gen.go` before/after)
- Any service code (Phase 1 only)

## Example: Before and After (AgentManagerAgent)

**Before:**
```yaml
AgentManagerAgent:
  type: object
  description: Represents an agent resource.
  properties:
    id:
      type: string
    customer_id:
      type: string
      description: Resource's customer ID.
    username:
      type: string
      description: Agent's username.
    name:
      type: string
      description: Agent's name.
    tag_ids:
      type: array
      items:
        type: string
      description: Agent's tag IDs.
    tm_create:
      type: string
      description: Created timestamp.
```

**After:**
```yaml
AgentManagerAgent:
  type: object
  description: Represents an agent resource.
  properties:
    id:
      type: string
      format: uuid
      x-go-type: string
      description: "The unique identifier of the agent. Returned from the `POST /agents` or `GET /agents` response."
      example: "550e8400-e29b-41d4-a716-446655440000"
    customer_id:
      type: string
      format: uuid
      x-go-type: string
      description: "The unique identifier of the customer who owns this agent. Returned from the `GET /customers` response."
      example: "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
    username:
      type: string
      description: "Agent's login username, typically an email address."
      example: "agent@example.com"
    name:
      type: string
      description: "Display name of the agent."
      example: "John Smith"
    tag_ids:
      type: array
      items:
        type: string
        format: uuid
        x-go-type: string
      description: "List of tag IDs assigned to this agent. Returned from the `POST /tags` or `GET /tags` response."
      example: ["b1a2c3d4-e5f6-7890-abcd-ef1234567890"]
    tm_create:
      type: string
      format: date-time
      x-go-type: string
      description: "Timestamp when the agent was created."
      example: "2026-01-15T09:30:00Z"
```

## Risks

1. **`x-go-type` might not be recognized by non-oapi-codegen tools** — Swagger UI and other tools will ignore this extension and show `format: uuid` as the canonical type. This is fine — it's an oapi-codegen-specific directive.
2. **Some documentation tools may not show examples alongside `$ref`** — Mitigated by also adding examples to enum schema definitions.
3. **Phase 2 (oneOf) may be more complex than estimated** — Will require its own design doc with detailed investigation of oapi-codegen union type generation.
