# AI-Native OpenAPI Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Apply Rules 2-5 of the AI-Native OpenAPI guidelines to every schema in `openapi.yaml` — adding `format:`, `x-go-type:`, `example:`, provenance descriptions, and `minItems:` — with zero breaking changes to generated Go types.

**Architecture:** All changes are in a single file (`bin-openapi-manager/openapi/openapi.yaml`). Each task edits a batch of schema groups in-place. After all edits, `go generate` is run in both `bin-openapi-manager` and `bin-api-manager` to verify no generated types changed.

**Tech Stack:** OpenAPI 3.0 YAML, oapi-codegen v2.5.1 (x-go-type extension)

**Working directory:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update`

---

## Transformation Patterns (Reference for All Tasks)

Every task applies these same patterns. Memorize these before starting.

### Pattern A: UUID Fields
```yaml
# BEFORE
id:
  type: string

# AFTER
id:
  type: string
  format: uuid
  x-go-type: string
  description: "<provenance description>"
  example: "550e8400-e29b-41d4-a716-446655440000"
```

### Pattern B: Timestamp Fields
```yaml
# BEFORE
tm_create:
  type: string
  description: "Timestamp when created."

# AFTER
tm_create:
  type: string
  format: date-time
  x-go-type: string
  description: "Timestamp when created."
  example: "2026-01-15T09:30:00Z"
```

### Pattern C: Free-Text Fields
```yaml
# BEFORE
name:
  type: string

# AFTER
name:
  type: string
  description: "<meaningful description>"
  example: "<realistic value>"
```

### Pattern D: Enum Schemas (add example to the enum definition itself)
```yaml
# BEFORE
SomeStatus:
  type: string
  description: Status
  enum:
    - active
    - inactive

# AFTER
SomeStatus:
  type: string
  description: Status
  example: "active"
  enum:
    - active
    - inactive
```

### Pattern E: $ref Fields (add description/example alongside for AI readability)
```yaml
# BEFORE
status:
  $ref: '#/components/schemas/SomeStatus'

# AFTER
status:
  $ref: '#/components/schemas/SomeStatus'
  description: "Current status of the resource."
  example: "active"
```

### Pattern F: Arrays Requiring Items
```yaml
# BEFORE
destinations:
  type: array
  items:
    $ref: '#/components/schemas/CommonAddress'

# AFTER
destinations:
  type: array
  items:
    $ref: '#/components/schemas/CommonAddress'
  minItems: 1
  description: "List of target addresses. Must contain at least one destination."
```

### Pattern G: UUID items in arrays
```yaml
# BEFORE
tag_ids:
  type: array
  items:
    type: string

# AFTER
tag_ids:
  type: array
  items:
    type: string
    format: uuid
    x-go-type: string
  description: "<provenance>"
  example: ["b1a2c3d4-e5f6-7890-abcd-ef1234567890"]
```

### Provenance Description Templates
Use these exact patterns for common ID fields:

| Field | Provenance Description |
|-------|----------------------|
| `id` (self) | "The unique identifier of the {resource}." |
| `customer_id` | "The unique identifier of the customer who owns this resource. Returned from the `GET /customers` response." |
| `owner_id` | "The unique identifier of the resource owner." |
| `flow_id` | "The unique identifier of the flow definition. Returned from the `POST /flows` or `GET /flows` response." |
| `activeflow_id` | "The unique identifier of the active flow instance. Returned from the `POST /activeflows` or `GET /activeflows` response." |
| `campaign_id` | "The unique identifier of the campaign. Returned from the `POST /campaigns` or `GET /campaigns` response." |
| `queue_id` | "The unique identifier of the queue. Returned from the `POST /queues` or `GET /queues` response." |
| `outdial_id` | "The unique identifier of the outdial. Returned from the `POST /outdials` or `GET /outdials` response." |
| `outplan_id` | "The unique identifier of the outplan. Returned from the `POST /outplans` or `GET /outplans` response." |
| `recording_id` | "The unique identifier of the recording. Returned from the `GET /recordings` response." |
| `transcribe_id` | "The unique identifier of the transcribe session. Returned from the `GET /transcribes` response." |
| `conference_id` | "The unique identifier of the conference. Returned from the `GET /conferences` response." |
| `reference_id` | "The unique identifier of the referenced resource." |
| `account_id` | "The unique identifier of the billing account. Returned from the `GET /billing_accounts/{id}` response." |
| `master_call_id` | "The unique identifier of the master call that initiated this call." |
| `groupcall_id` | "The unique identifier of the group call. Returned from the `POST /groupcalls` or `GET /groupcalls` response." |
| `ai_id` | "The unique identifier of the AI configuration. Returned from the `POST /ais` or `GET /ais` response." |
| `confbridge_id` | "The unique identifier of the conference bridge." |
| `billing_account_id` | "The unique identifier of the billing account. Returned from the `GET /billing_accounts/{id}` response." |
| `provider_id` | "The unique identifier of the provider. Returned from the `GET /providers` response." |

### Example UUIDs (Use Different UUIDs Per Schema to Look Realistic)
- Self `id`: `"550e8400-e29b-41d4-a716-446655440000"`
- `customer_id`: `"7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"`
- `flow_id`: `"a1b2c3d4-e5f6-7890-1234-567890abcdef"`
- `activeflow_id`: `"d4e5f6a7-b8c9-0123-4567-890abcdef012"`
- `campaign_id`: `"c3d4e5f6-a7b8-9012-3456-7890abcdef01"`
- `recording_id`: `"e5f6a7b8-c9d0-1234-5678-90abcdef0123"`
- `reference_id`: `"f6a7b8c9-d0e1-2345-6789-0abcdef01234"`
- `account_id`: `"b8c9d0e1-f2a3-4567-8901-23456789abcd"`
- `queue_id`: `"1a2b3c4d-5e6f-7890-abcd-ef1234567890"`
- `outplan_id`: `"2b3c4d5e-6f7a-8901-bcde-f12345678901"`
- `outdial_id`: `"3c4d5e6f-7a8b-9012-cdef-012345678901"`
- `master_call_id`: `"4d5e6f7a-8b9c-0123-def0-123456789012"`
- `groupcall_id`: `"5e6f7a8b-9c0d-1234-ef01-234567890123"`
- `ai_id`: `"6f7a8b9c-0d1e-2345-f012-345678901234"`
- `conference_id`: `"8b9c0d1e-2f3a-4567-0123-456789abcdef"`
- `transcribe_id`: `"9c0d1e2f-3a4b-5678-1234-567890abcdef"`
- `confbridge_id`: `"0d1e2f3a-4b5c-6789-2345-678901abcdef"`
- `tag_ids` item: `"b1a2c3d4-e5f6-7890-abcd-ef1234567890"`
- `provider_id`: `"a0b1c2d3-e4f5-6789-0abc-def012345678"`
- `owner_id`: `"c2d3e4f5-a6b7-8901-2cde-f01234567890"`

---

## Task 1: Baseline Snapshot

**Files:**
- Read: `bin-openapi-manager/gens/models/gen.go`
- Read: `bin-api-manager/gens/openapi_server/gen.go`

**Step 1: Save checksums of generated files before any changes**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update
md5sum bin-openapi-manager/gens/models/gen.go > /tmp/openapi-gen-before.md5
md5sum bin-api-manager/gens/openapi_server/gen.go >> /tmp/openapi-gen-before.md5
cat /tmp/openapi-gen-before.md5
```

Expected: Two md5 checksums printed. Save these — they will be compared after all edits.

**Step 2: No commit needed for this step.**

---

## Task 2: Update Components/Parameters + Auth + AgentManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~201-338

**Step 1: Update Components/Parameters (lines 203-217)**

Add `example:` to PageSize and PageToken. Update PageToken description.

```yaml
  parameters:
    PageSize:
      name: page_size
      in: query
      description: "Number of results to return per page."
      required: false
      schema:
        type: integer
        example: 25
    PageToken:
      name: page_token
      in: query
      description: "Cursor token for pagination. Use the `next_page_token` value from the previous response."
      required: false
      schema:
        type: string
        format: date-time
        x-go-type: string
        example: "2026-01-15T09:30:00Z"
```

**Step 2: Update Auth schemas (lines 225-239)**

AuthLoginResponse already has examples — verify and leave as-is.

**Step 3: Update AgentManager enum schemas (lines 244-294)**

Add `example:` to each enum definition:

- `AgentManagerAgentPermission`: add `example: 64`  (already has it — verify)
- `AgentManagerAgentRingMethod`: add `example: "ringall"`, fix description to `"Method used to ring the agent for incoming calls."`
- `AgentManagerAgentStatus`: add `example: "available"`

**Step 4: Update AgentManagerAgent (lines 296-338)**

Apply patterns A, B, C, E, G to every field:

- `id`: Pattern A — uuid + provenance "The unique identifier of the agent."
- `customer_id`: Pattern A — uuid + provenance
- `username`: Pattern C — example `"agent@example.com"`
- `name`: Pattern C — example `"John Smith"`
- `detail`: Pattern C — example `"Senior support agent"`
- `ring_method`: Pattern E — description + example alongside $ref
- `status`: Pattern E
- `permission`: Pattern E
- `tag_ids`: Pattern G — uuid items
- `addresses`: keep $ref array, add description
- `tm_create`, `tm_update`, `tm_delete`: Pattern B

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to Parameters, Auth, AgentManager schemas"
```

---

## Task 3: Update BillingManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~343-548

**Step 1: Update BillingManager enum schemas**

Add `example:` to each:
- `BillingManagerAccountPaymentMethod`: example `"credit card"`
- `BillingManagerAccountPaymentType`: example `"prepaid"`
- `BillingManagerAccountPlanType`: example `"basic"`
- `BillingManagerBillingreferenceType`: example `"call"`
- `BillingManagerBillingStatus`: example `"progressing"`
- `BillingManagerBillingTransactionType`: example `"usage"`

**Step 2: Update BillingManagerAccount (lines 378-421)**

Apply patterns to all fields:
- `id`, `customer_id`: Pattern A
- `name`: Pattern C — example `"Production Account"`
- `detail`: Pattern C — example `"Main billing account for production services"`
- `plan_type`: Pattern E
- `balance_credit`: add example `1500000` (1.50 USD in micros)
- `balance_token`: add example `500`
- `payment_type`, `payment_method`: Pattern E
- All `tm_*` fields: Pattern B

**Step 3: Update BillingManagerBilling (lines 476-547)**

Apply patterns to all fields:
- `id`, `customer_id`, `account_id`, `reference_id`: Pattern A with provenance
- `transaction_type`, `status`, `reference_type`: Pattern E
- `cost_type`: Pattern C — example `"call_pstn_outgoing"`
- `usage_duration`: example `125`
- `billable_units`: example `3`
- `rate_token_per_unit`: example `10`
- `rate_credit_per_unit`: example `50000`
- `amount_token`: example `-30`
- `amount_credit`: example `-150000`
- `balance_token_snapshot`: example `470`
- `balance_credit_snapshot`: example `1350000`
- `idempotency_key`: Pattern C — example `"txn-2026-01-15-001"`
- All `tm_*` fields: Pattern B

**Step 4: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to BillingManager schemas"
```

---

## Task 4: Update CallManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~553-903

**Step 1: Update CallManager enum schemas (lines 553-638)**

Add `example:` to each:
- `CallManagerCallDirection`: example `"outgoing"`
- `CallManagerCallHangupBy`: example `"remote"`
- `CallManagerCallHangupReason`: example `"normal"`
- `CallManagerCallMuteDirection`: example `"both"`
- `CallManagerCallStatus`: example `"progressing"`
- `CallManagerCallType`: example `"flow"`

**Step 2: Update CallManagerCall (lines 639-714)**

Apply patterns:
- `id`, `customer_id`, `owner_id`, `flow_id`, `activeflow_id`, `master_call_id`, `recording_id`, `groupcall_id`: Pattern A with provenance
- `owner_type`: Pattern C — example `"agent"`
- `chained_call_ids`: Pattern G
- `recording_ids`: Pattern G
- All `$ref` fields: Pattern E
- All `tm_*` fields: Pattern B

**Step 3: Update CallManagerGroupcall enums + schema (lines 716-824)**

Add examples to enums:
- `CallManagerGroupcallAnswerMethod`: example `"hangup_others"`
- `CallManagerGroupcallRingMethod`: example `"ring_all"`
- `CallManagerGroupcallStatus`: example `"progressing"`

Update CallManagerGroupcall fields:
- `id`, `customer_id`, `owner_id`, `flow_id`, `master_call_id`, `master_groupcall_id`, `answer_call_id`, `answer_groupcall_id`: Pattern A
- `owner_type`: Pattern C
- `destinations`: Pattern F (minItems: 1) + existing $ref
- `call_ids`, `groupcall_ids`: Pattern G
- `call_count`, `groupcall_count`, `dial_index`: add examples
- All `$ref` fields: Pattern E
- All `tm_*` fields: Pattern B

**Step 4: Update CallManagerRecording enums + schema (lines 826-902)**

Add examples to enums:
- `CallManagerRecordingFormat`: example `"wav"`
- `CallManagerRecordingReferenceType`: example `"call"`
- `CallManagerRecordingStatus`: example `"recording"`

Update CallManagerRecording fields — same patterns as above.

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to CallManager schemas (Call, Groupcall, Recording)"
```

---

## Task 5: Update CampaignManager + AIManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~907-1413

**Step 1: Update CampaignManager schemas (lines 907-1119)**

Enums — add `example:`:
- `CampaignManagerCampaignEndHandle`: `"stop"`
- `CampaignManagerCampaignExecute`: `"run"`
- `CampaignManagerCampaignStatus`: `"run"`
- `CampaignManagerCampaignType`: `"call"`
- `CampaignManagerCampaigncallReferenceType`: `"call"`
- `CampaignManagerCampaigncallResult`: `"success"`
- `CampaignManagerCampaigncallStatus`: `"progressing"`

CampaignManagerCampaign fields:
- `id`, `customer_id`, `outplan_id`, `outdial_id`, `queue_id`, `next_campaign_id`: Pattern A with provenance
- `name`: example `"Q1 Outbound Campaign"`
- `detail`: example `"Quarterly customer outreach campaign"`
- `service_level`: example `80`
- `actions`: array of FlowManagerAction — add description
- All `$ref` and `tm_*` fields: Patterns E, B

CampaignManagerCampaigncall and CampaignManagerOutplan — same patterns.
- `destination_index`: example `0`
- `try_count`: example `2`
- `dial_timeout`: example `30000`
- `try_interval`: example `60000`
- `max_try_count_0` through `max_try_count_4`: examples `3`, `2`, `1`, `1`, `1`

**Step 2: Update AIManager schemas (lines 1125-1413)**

Enums — add `example:`:
- `AIManagerAIEngineType`: `"chatGPT"`
- `AIManagerAIEngineModel`: `"gpt-4o"`
- `AIManagerAIcallGender`: `"female"`
- `AIManagerAIcallReferenceType`: `"call"`
- `AIManagerAIcallStatus`: `"progressing"`
- `AIManagerMessageDirection`: `"incoming"`
- `AIManagerMessageRole`: `"assistant"`
- `AIManagerSummaryReferenceType`: `"call"`
- `AIManagerSummaryStatus`: `"progressing"`

AIManagerAI fields:
- `id`, `customer_id`: Pattern A
- `name`: example `"Customer Support Bot"`
- `detail`: example `"AI assistant for handling customer inquiries"`
- `engine_type`, `engine_model`: Pattern E
- `engine_data`: KEEP as `additionalProperties: true` per design decision 4
- `engine_key`: example `"sk-...redacted..."`, description `"API key or authentication key for the AI engine. Write-only; not returned in responses."`
- `init_prompt`: example `"You are a helpful customer support assistant."`
- `tts_type`: example `"google"`, description `"Text-to-speech provider type."`
- `tts_voice_id`: example `"en-US-Neural2-F"`, description `"Text-to-speech voice identifier."`
- `stt_type`: example `"google"`, description `"Speech-to-text provider type."`
- `tool_names`: example `["all"]`, description already good
- All `tm_*`: Pattern B

AIManagerAIcall, AIManagerMessage, AIManagerSummary — same patterns.

**Step 3: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to CampaignManager and AIManager schemas"
```

---

## Task 6: Update Common + ConferenceManager + ContactManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~1418-1761

**Step 1: Update CommonAddress (lines 1419-1456)**

- `type` enum: add `example: "tel"`
- `target`: add `description: "The address endpoint. Format depends on type: phone number for tel (e.g. +14155551234), UUID for agent/conference/extension, email for email, SIP URI for sip."`, example `"+14155551234"`
- `target_name`: example `"John Smith"`
- `name`: example `"Main Office"`
- `detail`: example `"Primary contact number"`

**Step 2: Update CommonPagination (lines 1457-1462)**

- `next_page_token`: add `format: date-time`, `x-go-type: string`, example `"2026-01-15T09:30:00Z"`

**Step 3: Update ConferenceManager schemas (lines 1468-1611)**

Enums — add `example:`:
- `ConferenceManagerConferenceStatus`: `"progressing"`
- `ConferenceManagerConferenceType`: `"conference"`
- `ConferenceManagerConferencecallReferenceType`: `"call"`
- `ConferenceManagerConferencecallStatus`: `"joined"`

ConferenceManagerConference fields:
- `id`, `customer_id`, `pre_flow_id`, `post_flow_id`, `recording_id`, `transcribe_id`: Pattern A
- `name`: example `"Team Standup"`
- `detail`: example `"Daily team standup meeting"`
- `data`: KEEP as `additionalProperties: true` per design decision 4, add `description: "Custom key-value data associated with the conference."`
- `timeout`: example `3600`
- `conferencecall_ids`, `recording_ids`, `transcribe_ids`: Pattern G
- All `$ref` and `tm_*`: Patterns E, B

ConferenceManagerConferencecall — same patterns.

**Step 4: Update ContactManager schemas (lines 1617-1761)**

ContactManager is already partially compliant (has `format: uuid` and `format: date-time`). Fill remaining gaps:
- Add `example:` to all fields that don't have them
- Add `x-go-type: string` to all `format: uuid` fields (CRITICAL — without this, Go type is `*openapi_types.UUID`)
- Add `x-go-type: string` to all `format: date-time` fields (CRITICAL — without this, Go type is `*time.Time`)
- Add provenance descriptions to ID fields
- ContactManagerContactSource enum: add `example: "manual"`
- ContactManagerEmailType enum: add `example: "work"`
- ContactManagerPhoneNumberType enum: add `example: "mobile"`

**IMPORTANT for ContactManager:** This section already has `format: uuid` WITHOUT `x-go-type: string`, meaning the generated Go types are ALREADY `*openapi_types.UUID` and `*time.Time`. Adding `x-go-type: string` here would CHANGE the generated types (from UUID to string). Do NOT add `x-go-type: string` to ContactManager — leave the existing format as-is. Only add missing `example:` values and descriptions.

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to Common, ConferenceManager, ContactManager schemas"
```

---

## Task 7: Update ConversationManager + CustomerManager + EmailManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~1766-2197

**Step 1: Update ConversationManager schemas (lines 1766-1984)**

Enums — add `example:`:
- `ConversationManagerAccountType`: `"sms"`
- `ConversationManagerConversationReferenceType`: `"message"`
- `ConversationManagerMediaType`: `"image"`
- `ConversationManagerMessageDirection`: `"incoming"`
- `ConversationManagerMessageReferenceType`: `"call"`
- `ConversationManagerMessageStatus`: `"sent"`

All object schemas: apply standard patterns (A, B, C, E, G) to every field.

Notable fields:
- `ConversationManagerAccount.secret`: example `"whsec_...redacted..."`, description `"Webhook secret for signature verification. Write-only."`
- `ConversationManagerAccount.token`: example `"xoxb-...redacted..."`, description `"API token for the messaging platform. Write-only."`
- `ConversationManagerConversation.participants`: array of CommonAddress, add description
- `ConversationManagerMessage.text`: example `"Hello, how can I help you today?"`
- `ConversationManagerMedia.filename`: example `"photo_2026-01-15.jpg"`

**Step 2: Update CustomerManager schemas (lines 1990-2085)**

Enums — add `example:`:
- `CustomerManagerCustomerWebhookMethod`: `"POST"`
- `CustomerManagerCustomerStatus`: `"active"`

CustomerManagerAccesskey — add patterns to ALL fields (currently has zero descriptions/examples):
- `id`, `customer_id`: Pattern A
- `name`: example `"Production API Key"`
- `detail`: example `"API key for production environment"`
- `token`: example `"ak_live_...redacted..."`, description `"The access key token. Only returned once at creation time."`
- `tm_expire`: Pattern B
- All other `tm_*`: Pattern B

CustomerManagerCustomer fields:
- `id`: Pattern A
- `name`: example `"Acme Corporation"`
- `detail`: example `"Enterprise customer account"`
- `email`: add `format: email`, `x-go-type: string`, example `"admin@acme.com"`
- `phone_number`: example `"+14155551234"`, description `"Customer's contact phone number in E.164 format."`
- `address`: example `"123 Main St, San Francisco, CA 94105"`
- `webhook_method`, `status`: Pattern E
- `webhook_uri`: example `"https://api.acme.com/webhooks/voipbin"`, description `"URI where webhook events are delivered."`
- `billing_account_id`: Pattern A with provenance
- `email_verified`: example `true`
- `tm_deletion_scheduled`: Pattern B, description `"Timestamp when account deletion was requested. Null if not scheduled."`
- All `tm_*`: Pattern B

**Step 3: Update EmailManager schemas (lines 2090-2196)**

Enums — add `example:`:
- `EmailManagerEmailAttachmentReferenceType`: `"recording"`
- `EmailManagerEmailStatus`: `"delivered"`

EmailManagerEmailAttachment:
- `reference_id`: Pattern A

EmailManagerEmail:
- `id`, `customer_id`, `activeflow_id`: Pattern A
- `destinations`: Pattern F (minItems: 1)
- `subject`: example `"Your Call Recording is Ready"`
- `content`: example `"Please find your call recording attached."`
- `attachments`: add description
- All `tm_*`: Pattern B

**Step 4: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to ConversationManager, CustomerManager, EmailManager schemas"
```

---

## Task 8: Update FlowManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~2200-2896

This is the largest section. The `FlowManagerAction.option` field stays as `additionalProperties: true` (Phase 2 scope). All ActionOption sub-schemas get examples and descriptions.

**Step 1: Update FlowManagerActionType enum (lines 2201-2285)**

Add `example: "talk"`.

**Step 2: Update all FlowManagerActionOption* schemas (lines 2287-2708)**

For each ActionOption schema, add `example:` and `description:` to every field. Key examples:

- `FlowManagerActionOptionAITalk.ai_id`: Pattern A, description fix to `"The unique identifier of the AI configuration to use."`
- `FlowManagerActionOptionAITalk.language`: example `"en-US"`
- `FlowManagerActionOptionAITalk.duration`: example `300`
- `FlowManagerActionOptionCall.flow_id`: Pattern A
- `FlowManagerActionOptionCall.destinations`: Pattern F (minItems: 1)
- `FlowManagerActionOptionConnect.destinations`: Pattern F (minItems: 1)
- `FlowManagerActionOptionConversationSend.conversation_id`: Pattern A, example with provenance
- `FlowManagerActionOptionDigitsReceive.duration`: example `10000`
- `FlowManagerActionOptionDigitsReceive.key`: example `"#"`
- `FlowManagerActionOptionDigitsReceive.length`: example `4`
- `FlowManagerActionOptionDigitsSend.digits`: example `"1234#"`
- `FlowManagerActionOptionDigitsSend.duration`: example `250`
- `FlowManagerActionOptionDigitsSend.interval`: example `100`
- `FlowManagerActionOptionEcho.duration`: example `30000`
- `FlowManagerActionOptionEmailSend.destinations`: Pattern F (minItems: 1)
- `FlowManagerActionOptionEmailSend.subject`: example `"Meeting Recording"`
- `FlowManagerActionOptionEmailSend.content`: example `"Your recording is attached."`
- `FlowManagerActionOptionExternalMediaStart.external_host`: example `"media.example.com:10000"`
- `FlowManagerActionOptionFetch.event_url`: example `"https://api.example.com/webhooks/flow-event"`
- `FlowManagerActionOptionFetch.event_method`: example `"POST"`
- `FlowManagerActionOptionFetchFlow.flow_id`: Pattern A
- `FlowManagerActionOptionGoto.target_id`: Pattern A, description `"The action ID within the flow to jump to."`
- `FlowManagerActionOptionGoto.loop_count`: example `3`
- `FlowManagerActionOptionHangup.reason`: example `"normal"`
- `FlowManagerActionOptionHangup.reference_id`: Pattern A
- `FlowManagerActionOptionMessageSend.destinations`: Pattern F (minItems: 1)
- `FlowManagerActionOptionMessageSend.text`: example `"Your verification code is 1234."`
- `FlowManagerActionOptionPlay.stream_urls`: example `["https://media.voipbin.net/audio/greeting.wav"]`, add `minItems: 1`
- `FlowManagerActionOptionQueueJoin.queue_id`: Pattern A
- `FlowManagerActionOptionRecordingStart.format`: example `"wav"`
- `FlowManagerActionOptionRecordingStart.on_end_flow_id`: Pattern A
- `FlowManagerActionOptionSleep.duration`: example `5000`
- `FlowManagerActionOptionStreamEcho.duration`: example `30000`
- `FlowManagerActionOptionTalk.text`: example `"Welcome to VoIPBin. How can I help you?"`
- `FlowManagerActionOptionTalk.language`: example `"en-US"`
- `FlowManagerActionOptionTranscribeStart.language`: example `"en-US"`
- `FlowManagerActionOptionTranscribeStart.on_end_flow_id`: Pattern A
- `FlowManagerActionOptionTranscribeRecording.language`: example `"en-US"`
- `FlowManagerActionOptionVariableSet.key`: example `"caller_name"`
- `FlowManagerActionOptionVariableSet.value`: example `"John Smith"`
- `FlowManagerActionOptionWebhookSend.uri`: example `"https://api.example.com/webhooks"`
- `FlowManagerActionOptionWebhookSend.data_type`: example `"application/json"`
- `FlowManagerActionOptionConfbridgeJoin.confbridge_id`: Pattern A
- `FlowManagerActionOptionConferenceJoin.conference_id`: Pattern A

**Step 3: Update FlowManagerAction (lines 2718-2776)**

- `id`: Pattern A — description `"The unique identifier of this action within the flow."`
- `next_id`: Pattern A — description `"The identifier of the next action to execute. Null if this is the last action."`
- `option`: KEEP as `additionalProperties: true` — do NOT change (Phase 2)
- `tm_execute`: Pattern B

**Step 4: Update remaining FlowManager schemas (lines 2777-2896)**

Enums — add `example:`:
- `FlowManagerActiveflowStatus`: `"running"`
- `FlowManagerReferenceType`: `"call"`
- `FlowManagerFlowType`: `"flow"`

FlowManagerActiveflow:
- All ID fields: Pattern A with provenance
- `executed_actions`: add description
- All `tm_*`: Pattern B

FlowManagerFlow:
- `id`, `customer_id`, `on_complete_flow_id`: Pattern A
- `name`: example `"Inbound Call Handler"`
- `detail`: example `"Main flow for handling inbound customer calls"`
- `actions`: array, add description `"Ordered list of actions to execute in this flow."`
- All `tm_*`: Pattern B

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to FlowManager schemas (ActionOptions, Action, Activeflow, Flow)"
```

---

## Task 9: Update MessageManager + NumberManager + OutdialManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~2900-3226

**Step 1: Update MessageManager schemas (lines 2900-2998)**

Enums — add `example:`:
- `MessageManagerMessageDirection`: `"outbound"`
- `MessageManagerMessageProviderName`: `"telnyx"`
- `MessageManagerMessageType`: `"sms"`
- `MessageManagerTargetStatus`: `"delivered"`

MessageManagerMessage:
- `id`, `customer_id`: Pattern A
- `text`: example `"Your appointment is confirmed for tomorrow at 2 PM."`
- `targets`: array, add description
- All `$ref` and `tm_*`: Patterns E, B

MessageManagerTarget:
- `parts`: example `1`
- All `$ref` and `tm_*`: Patterns E, B

**Step 2: Update NumberManager schemas (lines 3001-3119)**

Enums — add `example:`:
- `NumberManagerAvailableNumber` (this is actually a feature enum): `"voice"`
- `NumberManagerNumberType`: `"normal"`
- `NumberManagerNumberProviderName`: `"telnyx"`
- `NumberManagerNumberStatus`: `"active"`

NumberManagerAvailableNumberFeature:
- `number`: example `"+14155551234"`, description `"The available phone number in E.164 format."`
- `country`: example `"US"`
- `region`: example `"California"`
- `postal_code`: example `"94105"`

NumberManagerNumber:
- `id`, `customer_id`, `call_flow_id`, `message_flow_id`: Pattern A with provenance
- `number`: example `"+14155551234"`, description `"The phone number in E.164 format (normal) or +899 format (virtual)."`
- `name`: example `"Main Support Line"`
- `detail`: example `"Primary inbound number for customer support"`
- All `$ref` and `tm_*`: Patterns E, B
- `t38_enabled`: example `false`
- `emergency_enabled`: example `false`

**Step 3: Update OutdialManager schemas (lines 3124-3225)**

OutdialManagerOutdial:
- `id`, `customer_id`, `campaign_id`: Pattern A
- `name`: example `"Q1 Target List"`
- `detail`: example `"Outbound dial list for Q1 campaign"`
- `data`: example `"custom-data"`, description `"Custom data associated with the outdial."`
- All `tm_*`: Pattern B

OutdialManagerOutdialtargetStatus: add `example: "idle"`

OutdialManagerOutdialtarget:
- `id`, `outdial_id`: Pattern A
- `name`: example `"John Smith"`
- `detail`: example `"Priority customer"`
- `data`: example `"vip-tag"`
- `destination_0` through `destination_4`: Pattern E with descriptions
- `try_count_0` through `try_count_4`: examples `0`
- All `tm_*`: Pattern B

**Step 4: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to MessageManager, NumberManager, OutdialManager schemas"
```

---

## Task 10: Update QueueManager + RegistrarManager + RouteManager + StorageManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~3228-3598

**Step 1: QueueManager schemas (lines 3228-3346)**

Enums — add `example:`:
- `QueueManagerQueueRoutingMethod`: `"random"`, add `description: "Method used to route incoming queue calls to available agents."`
- `QueueManagerQueuecallReferenceType`: `"call"`
- `QueueManagerQueuecallStatus`: `"waiting"`, add `description: "Lifecycle status of a queue call."`

QueueManagerQueue:
- `id`, `customer_id`, `wait_flow_id`: Pattern A
- `name`: example `"Customer Support Queue"`
- `detail`: example `"Main inbound support queue"`
- `tag_ids`: Pattern G
- `wait_queuecall_ids`, `service_queuecall_ids`: Pattern G
- `total_incoming_count`: example `150`
- `total_serviced_count`: example `120`
- `total_abandoned_count`: example `30`
- `wait_timeout`: example `300000`
- `service_timeout`: example `600000`
- All `tm_*`: Pattern B

QueueManagerQueuecall:
- `id`, `customer_id`, `reference_id`, `service_agent_id`: Pattern A
- `duration_waiting`: example `45000`
- `duration_service`: example `180000`
- All `$ref` and `tm_*`: Patterns E, B

**Step 2: RegistrarManager schemas (lines 3350-3427)**

RegistrarManagerAuthType: add `example: "basic"`

RegistrarManagerExtension:
- `id`, `customer_id`: Pattern A
- `name`: example `"Reception Desk"`
- `detail`: example `"Front desk extension"`
- `extension`: example `"1001"`, description `"The SIP extension number."`
- `domain_name`: example `"7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"`, description `"Domain name for SIP routing. Same as customer_id."`
- `username`: example `"1001"`, description `"SIP username. Same as the extension number."`
- `password`: example `"...redacted..."`, description `"SIP password. Write-only; not returned in responses."`
- `direct_hash`: example `"a1b2c3d4"`, description `"Hash for direct extension access via SIP URI sip:direct.<hash>@sip.voipbin.net."`
- All `tm_*`: Pattern B

RegistrarManagerTrunk:
- `id`, `customer_id`: Pattern A
- `name`: example `"Office PBX Trunk"`
- `detail`: example `"SIP trunk for office PBX system"`
- `domain_name`: example `"trunk.example.com"`
- `auth_types`: add description `"Authentication types enabled for this trunk."`
- `username`: example `"trunk_user"`
- `password`: example `"...redacted..."`
- `allowed_ips`: example `["203.0.113.10", "198.51.100.20"]`
- All `tm_*`: Pattern B

**Step 3: RouteManager schemas (lines 3430-3510)**

RouteManagerProviderType: add `example: "sip"`

RouteManagerProvider:
- `id`: Pattern A
- `hostname`: example `"sip.provider.com"`, description `"The SIP destination hostname for this provider."`
- `tech_prefix`: example `"001"`
- `tech_postfix`: example `""`
- `tech_headers`: add `description: "Custom SIP headers to include in requests to this provider."`
- `name`: example `"Primary SIP Provider"`
- `detail`: example `"Main PSTN termination provider"`
- All `tm_*`: Pattern B

RouteManagerRoute:
- `id`, `customer_id`, `provider_id`: Pattern A with provenance
- `name`: example `"US Domestic Route"`
- `detail`: example `"Route for US domestic calls"`
- `priority`: example `10`
- `target`: example `"+1"`, description `"Destination pattern for route matching (e.g., country code '+1' or 'all' for default)."`
- All `tm_*`: Pattern B

**Step 4: StorageManager schemas (lines 3514-3598)**

StorageManagerFileReferenceType: add `example: "recording"`

StorageManagerAccount:
- `id`, `customer_id`: Pattern A
- `total_file_count`: example `42`
- `total_file_size`: example `1073741824` (1 GB)
- All `tm_*`: Pattern B

StorageManagerFile:
- `id`, `customer_id`, `owner_id`, `reference_id`: Pattern A
- `name`: example `"Call Recording 2026-01-15"`
- `detail`: example `"Recording of customer support call"`
- `filename`: example `"recording-550e8400.wav"`
- `filesize`: example `2097152` (2 MB)
- `uri_download`: example `"https://storage.voipbin.net/files/550e8400/download"`, description `"Temporary presigned URL for downloading the file."`
- `tm_download_expire`: Pattern B, description `"Timestamp when the download URL expires."`
- All other `tm_*`: Pattern B

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to QueueManager, RegistrarManager, RouteManager, StorageManager schemas"
```

---

## Task 11: Update TalkManager + TimelineManager + TagManager + TtsManager + TranscribeManager + TransferManager + RagManager Schemas

**File:** `bin-openapi-manager/openapi/openapi.yaml`
**Lines:** ~3600-4078

**Step 1: TalkManager schemas (lines 3604-3790)**

Enums — add `example:`:
- `TalkManagerTalkType`: `"direct"`
- `TalkManagerMessageType`: `"normal"`
- `TalkManagerMediaType`: `"file"`

TalkManagerParticipantInput:
- `owner_type`: example `"agent"`
- `owner_id`: Pattern A

TalkManagerTalk:
- `id`, `customer_id`: Pattern A
- `name`: example `"Support Chat"`
- `detail`: example `"Customer support conversation"`
- `member_count`: example `2`
- All `$ref` and `tm_*`: Patterns E, B

TalkManagerMessage:
- `id`, `customer_id`, `owner_id`, `chat_id`, `parent_id`: Pattern A
- `owner_type`: example `"agent"`
- `text`: example `"I can help you with that. Let me check your account."`
- `medias`: add description
- All `$ref` and `tm_*`: Patterns E, B

TalkManagerMedia:
- `agent_id`: already has `format: uuid` — do NOT add `x-go-type: string` (check if it already generates as UUID)
- `file_id`: Pattern A
- `link_url`: already has `format: uri` — add example `"https://docs.voipbin.net/guides/getting-started"`

TalkManagerReaction:
- `emoji`: example `"👍"`
- `owner_type`: example `"agent"`
- `owner_id`: Pattern A
- `tm_create`: Pattern B

TalkManagerParticipant:
- `id`, `customer_id`, `owner_id`, `chat_id`: Pattern A
- `owner_type`: example `"agent"`
- `tm_joined`: Pattern B

**IMPORTANT for TalkManager:** Check `TalkManagerMedia.agent_id` — it already has `format: uuid` in the spec (line 3735). Do NOT add `x-go-type: string` unless you verify this field was already generating as UUID (which it likely is since format was already there). Same logic as ContactManager.

**Step 2: TimelineManager schema (lines 3795-3808)**

TimelineManagerEvent:
- `timestamp`: already has `format: date-time` — add example `"2026-01-15T09:30:00Z"`, do NOT add `x-go-type: string` (check existing type)
- `event_type`: example `"call_created"`, description `"Type of timeline event (e.g., call_created, conference_started, recording_ended)."`
- `data`: KEEP as generic `type: object` per design decision 4

**Step 3: TagManager schema (lines 3814-3834)**

TagManagerTag:
- `id`: Pattern A
- `name`: example `"VIP"`
- `detail`: example `"High-priority customer tag"`
- All `tm_*`: Pattern B

**Step 4: TtsManager schema (lines 3840-3881)**

TtsManagerSpeaking:
- `id`, `customer_id`, `reference_id`: Pattern A
- `reference_type`: example `"call"`, description `"Type of the referenced entity (call, confbridge)."`
- `language`: example `"en-US"`
- `provider`: example `"elevenlabs"`
- `voice_id`: example `"21m00Tcm4TlvDq8ikWAM"`
- `direction`: example `"both"`, description `"Audio injection direction (in, out, both)."`
- `status`: example `"active"`, description `"Session status (initiating, active, stopped)."`
- `pod_id`: example `"tts-manager-7f8d9c-abc12"`, description `"Kubernetes pod hosting this session. Internal use only."`
- All `tm_*`: Pattern B

**Step 5: TranscribeManager schemas (lines 3886-3979)**

Enums — add `example:`:
- `TranscribeManagerTranscribeDirection`: `"both"`, add `description: "Direction of audio to transcribe."`
- `TranscribeManagerTranscribeReferenceType`: `"call"`, add `description: "Type of the resource being transcribed."`
- `TranscribeManagerTranscribeStatus`: `"progressing"`, add `description: "Current status of the transcription."`
- `TranscribeManagerTranscriptDirection`: `"both"`, add `description: "Direction of the transcribed audio segment."`

TranscribeManagerTranscribe:
- `id`, `customer_id`, `reference_id`: Pattern A
- `language`: example `"en-US"`
- All `$ref` and `tm_*`: Patterns E, B

TranscribeManagerTranscript:
- `id`, `transcribe_id`: Pattern A
- `message`: example `"Hello, I would like to check my account balance."`, description `"The transcribed text content."`
- `tm_transcript`: Pattern B, description `"Timestamp of when this segment was spoken."`
- `tm_create`: Pattern B

**Step 6: TransferManager schemas (lines 3984-4029)**

TransferManagerTransferType: add `example: "blind"`, `description: "Type of call transfer."`

TransferManagerTransfer:
- `id`, `customer_id`, `transferer_call_id`, `transferee_call_id`, `groupcall_id`, `confbridge_id`: Pattern A
- `transferee_addresses`: Pattern F (minItems: 1)
- All `tm_*`: Pattern B

**Step 7: RagManager schemas (lines 4034-4078)**

RagManagerDocType: add `example: "openapi"`, `description: "Type of documentation source for RAG retrieval."`

RagManagerSource:
- `source_file`: example `"docs/api-reference/calls.md"`
- `section_title`: example `"Creating a Call"`
- `relevance_score`: example `0.92`
- `doc_type`: Pattern E

RagManagerQueryResponse:
- `answer`: example `"To create a call, use the POST /calls endpoint with source and destination addresses."`
- `sources`: add description

**Step 8: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Apply AI-native rules to TalkManager, TimelineManager, TagManager, TtsManager, TranscribeManager, TransferManager, RagManager schemas"
```

---

## Task 12: Verification — Regenerate and Diff

**Step 1: Regenerate models in bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update/bin-openapi-manager
go generate ./...
```

Expected: No errors.

**Step 2: Verify generated types did NOT change**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update
md5sum bin-openapi-manager/gens/models/gen.go
```

Compare with the checksum from Task 1. The checksums WILL differ (because oapi-codegen includes descriptions in comments and examples may appear). But the STRUCT DEFINITIONS must be identical.

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update
git diff bin-openapi-manager/gens/models/gen.go | grep -E "^\+.*\*(string|int|bool|float|map|openapi_types|time\.Time|\[\])" | head -20
```

Expected: NO output. If there are type changes, something went wrong.

Also verify no new type imports appeared:
```bash
git diff bin-openapi-manager/gens/models/gen.go | grep -E "^\+.*import" | head -10
```

Expected: NO new imports (especially no new `openapi_types` or `time` imports that weren't there before).

**Step 3: Run openapi-manager verification**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update/bin-openapi-manager
go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 4: Regenerate and verify bin-api-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update/bin-api-manager
go generate ./...
go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 5: Commit generated files if changed**

If `go generate` updated `gen.go` files (descriptions in comments, etc.), commit them:

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update
git add bin-openapi-manager/gens/models/gen.go bin-api-manager/gens/openapi_server/gen.go
git commit -m "NOJIRA-AI-native-openapi-spec-update

- bin-openapi-manager: Regenerate models after AI-native spec updates
- bin-api-manager: Regenerate server code after AI-native spec updates"
```

---

## Task 13: Push and Create PR

**Step 1: Push branch**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-AI-native-openapi-spec-update
git push -u origin NOJIRA-AI-native-openapi-spec-update
```

**Step 2: Create PR**

Run:
```bash
gh pr create --title "NOJIRA-AI-native-openapi-spec-update" --body "$(cat <<'PREOF'
Apply AI-Native OpenAPI Specification Rules (2-5) to all schemas in openapi.yaml.
This makes the spec readable by AI agents by adding format specifiers, realistic
examples, provenance descriptions, and array constraints — with zero breaking
changes to generated Go types.

- bin-openapi-manager: Add format: uuid + x-go-type: string to all UUID fields
- bin-openapi-manager: Add format: date-time + x-go-type: string to all timestamp fields
- bin-openapi-manager: Add provenance descriptions to all reference ID fields
- bin-openapi-manager: Add realistic example values to all leaf properties
- bin-openapi-manager: Add example values to all enum schema definitions
- bin-openapi-manager: Add minItems: 1 to arrays requiring at least one item
- bin-openapi-manager: Regenerate Go models (no type changes)
- bin-api-manager: Regenerate server code (no type changes)
- docs: Add design document for AI-native OpenAPI spec update
PREOF
)"
```

---

## Checklist Summary

- [ ] Task 1: Baseline snapshot (checksums saved)
- [ ] Task 2: Parameters + Auth + AgentManager
- [ ] Task 3: BillingManager
- [ ] Task 4: CallManager (Call, Groupcall, Recording)
- [ ] Task 5: CampaignManager + AIManager
- [ ] Task 6: Common + ConferenceManager + ContactManager
- [ ] Task 7: ConversationManager + CustomerManager + EmailManager
- [ ] Task 8: FlowManager (ActionOptions, Action, Activeflow, Flow)
- [ ] Task 9: MessageManager + NumberManager + OutdialManager
- [ ] Task 10: QueueManager + RegistrarManager + RouteManager + StorageManager
- [ ] Task 11: TalkManager + TimelineManager + TagManager + TtsManager + TranscribeManager + TransferManager + RagManager
- [ ] Task 12: Verification (regenerate + diff + tests)
- [ ] Task 13: Push + PR
