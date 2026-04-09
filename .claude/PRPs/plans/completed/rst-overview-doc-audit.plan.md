# Plan: RST Overview Documentation Audit & Sync

## Summary
Audit all 41 `*_overview.rst` files against current API endpoints (OpenAPI spec), status/enum values (model definitions), and feature behavior (service handlers). Fix discrepancies, ensure AI-Native RST guidelines are followed, and rebuild Sphinx HTML. This is Phase 2 of the full RST documentation sync initiative.

## User Story
As an external developer or AI agent integrating with VoIPbin APIs,
I want overview documentation that accurately describes current API behavior, endpoints, and status values,
So that I can understand how each resource works without reading source code.

## Problem -> Solution
Overview docs may reference deprecated endpoints, missing status values, or outdated behavior descriptions -> Every overview doc will accurately reflect current API endpoints, status/enum values, and resource behavior.

## Metadata
- **Complexity**: Large
- **Source PRD**: `.claude/PRPs/prds/rst-docs-full-audit-sync.prd.md`
- **PRD Phase**: Phase 2 -- Overview doc audit
- **Estimated Files**: ~41 overview RST files + HTML rebuild
- **Confidence Score**: 7/10 (overview docs are prose-heavy, harder to systematically verify than struct docs)

---

## UX Design

N/A -- internal documentation change. No UI surfaces affected.

---

## Mandatory Reading

Files that MUST be read before implementing:

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `bin-api-manager/docsdev/source/accesskey_overview.rst` | all | Well-structured overview with AI Context + AI Hints (reference pattern) |
| P0 | `bin-api-manager/docsdev/source/webhook_overview.rst` | all | Good overview with troubleshooting section (reference pattern) |
| P0 | `bin-openapi-manager/openapi/openapi.yaml` | paths section | Source of truth for API endpoint existence |
| P1 | `bin-api-manager/CLAUDE.md` | AI-Native RST section | Rules for overview files (AI Context, AI Hints, enum listing) |
| P1 | Phase 1 report | all | Lessons learned from struct audit |

## External Documentation

No external research needed -- feature uses established internal patterns.

---

## Patterns to Mirror

### OVERVIEW_RST_FORMAT
// SOURCE: bin-api-manager/docsdev/source/accesskey_overview.rst
```rst
.. _resource-overview:

Resource API Overview
=====================

.. note:: **AI Context**

   * **Complexity:** Low | Medium | High
   * **Cost:** Free | Chargeable (credit deduction)
   * **Async:** Yes/No. If yes, state how to track status.

Description of the resource and what it does.

.. note:: **AI Implementation Hint**

   Non-obvious behavior, normalization rules, or gotchas.

Troubleshooting
---------------

* **400 Bad Request:**
    * **Cause:** ...
    * **Fix:** ...
```

### STATUS_ENUM_FORMAT
// SOURCE: bin-api-manager/docsdev/source/call_overview.rst
```rst
Status Lifecycle
----------------
* ``dialing``: System is currently dialing the destination.
* ``ringing``: Destination device is ringing.
* ``progressing``: Call answered. Audio flowing.
* ``hangup``: Call ended. Final state.
```

### ENDPOINT_URL_FORMAT
// SOURCE: bin-api-manager/CLAUDE.md (AI-Native RST rules)
```rst
Always use fully qualified URLs:
  GET https://api.voipbin.net/v1.0/calls
  POST https://api.voipbin.net/v1.0/calls

Never use:
  GET /calls
  POST /v1/calls
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| 41 `*_overview.rst` files | AUDIT/UPDATE | Verify endpoints, status values, behavior descriptions |
| ~278 HTML build files | REBUILD | Sphinx HTML must reflect RST changes |

## NOT Building

- New overview docs for undocumented resources (that's separate work)
- Rewriting overview prose style or structure (preserve existing patterns)
- Tutorial updates (Phase 3)
- Quickstart updates (Phase 4)
- OpenAPI spec changes (if docs are wrong, fix docs, not code)

---

## Step-by-Step Tasks

### Audit Methodology Per File

For each overview RST file:

1. **Read the overview RST file**
2. **Verify endpoints**: Compare every API endpoint URL mentioned against the OpenAPI spec (`bin-openapi-manager/openapi/openapi.yaml`). Check that:
   - Referenced endpoints exist
   - HTTP methods are correct
   - URLs use FQDN format (`https://api.voipbin.net/v1.0/...`)
   - No admin-only endpoints in user docs (`/customers/{id}` -> `/customer`)
3. **Verify status/enum values**: For resources with status fields, compare documented values against model definitions (`models/<entity>/status.go` or enum constants in the model files)
4. **Verify feature descriptions**: Check that behavioral descriptions match current service handler implementation
5. **Check AI-Native compliance**: Ensure AI Context block, at least one AI Implementation Hint, and troubleshooting sections exist
6. **Fix discrepancies**: Update RST to match current codebase

### Batching Strategy

Group by service domain for context efficiency:

### Task 2A: Core Resources
- **FILES**: `customer_overview.rst`, `agent_overview.rst`, `call_overview.rst`, `flow_overview.rst`, `activeflow_overview.rst`
- **ACTION**: Audit each file against OpenAPI endpoints and model status values
- **SOURCE OF TRUTH**:
  - `bin-customer-manager/models/customer/` -- customer status/fields
  - `bin-agent-manager/models/agent/` -- agent status/permission enums
  - `bin-call-manager/models/call/` -- call status lifecycle
  - `bin-flow-manager/models/flow/` -- flow status
  - `bin-flow-manager/models/activeflow/` -- activeflow status
- **VALIDATE**: All endpoints in these files exist in OpenAPI; all status values match model definitions
- **GOTCHA**: `call_overview.rst` is 784 lines -- most complex, likely has most drift

### Task 2B: Communication Resources
- **FILES**: `message_overview.rst`, `email_overview.rst`, `conversation_overview.rst`, `speaking_overview.rst`, `talk_overview.rst`
- **ACTION**: Audit each file
- **SOURCE OF TRUTH**:
  - `bin-message-manager/models/message/` -- message status
  - `bin-email-manager/models/email/` -- email status
  - `bin-conversation-manager/models/conversation/` -- conversation type/status (MAJOR rewrite in Phase 1)
  - `bin-call-manager/models/speaking/` -- speaking status
  - `bin-talk-manager/models/talk/` -- talk status
- **GOTCHA**: `conversation_overview.rst` likely needs significant updates given the struct was majorly rewritten in Phase 1

### Task 2C: Campaign & Queue Resources
- **FILES**: `campaign_overview.rst`, `queue_overview.rst`, `outdial_overview.rst`, `outplan_overview.rst`
- **ACTION**: Audit each file
- **SOURCE OF TRUTH**:
  - `bin-campaign-manager/models/campaign/` -- campaign status
  - `bin-queue-manager/models/queue/` -- queue/queuecall status
  - `bin-outdial-manager/models/outdial/` -- outdial status
  - `bin-outdial-manager/models/outplan/` -- outplan status
- **VALIDATE**: Queue and campaign status lifecycles documented correctly

### Task 2D: Infrastructure Resources
- **FILES**: `number_overview.rst`, `route_overview.rst`, `extension_overview.rst`, `recording_overview.rst`, `storage_overview.rst`, `tag_overview.rst`, `webhook_overview.rst`, `websocket_overview.rst`, `provider_overview.rst`, `trunk_overview.rst`, `trunk_overview_trunking.rst`, `trunk_overview_domain_name.rst`
- **ACTION**: Audit each file
- **SOURCE OF TRUTH**:
  - `bin-number-manager/models/number/` -- number status
  - `bin-route-manager/models/route/` -- route fields
  - `bin-registrar-manager/models/extension/` -- extension status
  - `bin-call-manager/models/recording/` -- recording status
  - `bin-storage-manager/models/file/` -- storage file status
  - `bin-tag-manager/models/tag/` -- tag fields
  - `bin-webhook-manager/models/webhook/` -- webhook fields
  - `bin-registrar-manager/models/trunk/` -- trunk status
- **GOTCHA**: `recording_overview.rst` had significant struct changes in Phase 1 -- overview likely stale too

### Task 2E: AI & Advanced Resources
- **FILES**: `ai_overview.rst`, `ai_voice_agent_integration_overview.rst`, `rag_overview.rst`, `transcribe_overview.rst`, `conference_overview.rst`, `mediastream_overview.rst`
- **ACTION**: Audit each file
- **SOURCE OF TRUTH**:
  - `bin-ai-manager/models/ai/` -- AI status
  - `bin-ai-manager/models/aicall/` -- AI call status
  - `bin-rag-manager/models/rag/` -- RAG fields
  - `bin-transcribe-manager/models/transcribe/` -- transcribe status
  - `bin-conference-manager/models/conference/` -- conference status (significant Phase 1 changes)
  - Various models for streaming/transcript
- **GOTCHA**: `conference_overview.rst` at 759 lines -- conference struct had major changes in Phase 1

### Task 2F: Meta/System Resources
- **FILES**: `accesskey_overview.rst`, `team_overview.rst`, `billing_account_overview.rst`, `contact_overview.rst`, `common_overview.rst`, `variable_overview.rst`, `sdk_overview.rst`, `direct_hash_overview.rst`, `architecture_overview.rst`
- **ACTION**: Audit each file
- **SOURCE OF TRUTH**:
  - `bin-agent-manager/models/accesskey/` -- accesskey fields
  - `bin-agent-manager/models/team/` -- team fields
  - `bin-billing-manager/models/billing/` -- billing status
  - `bin-contact-manager/models/contact/` -- contact fields
  - Architecture docs -- verify against current service topology
- **GOTCHA**: `architecture_overview.rst` at 409 lines -- may reference outdated service names/topology. `billing_account_overview.rst` -- Phase 1 added billing struct, overview may need billing type references

### Task G: Sphinx HTML Rebuild
- **ACTION**: Clean rebuild all HTML
- **IMPLEMENT**: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`
- **VALIDATE**: Zero errors, zero warnings
- **GOTCHA**: Force-add build output: `git add -f bin-api-manager/docsdev/build/`

---

## Testing Strategy

### Validation per File

| Check | Method | Expected |
|---|---|---|
| Endpoints exist | Compare against OpenAPI paths | All referenced endpoints found |
| Status values correct | Compare against model definitions | All status values match |
| AI Context block present | Grep for `AI Context` | Present in every overview |
| AI Implementation Hint present | Grep for `AI Implementation Hint` | At least one per overview |
| FQDN URLs used | Grep for endpoint format | All use `https://api.voipbin.net/v1.0/` |
| No admin endpoints | Grep for `/customers/` | None in user-facing docs |

### Edge Cases Checklist
- [ ] Files with no API endpoints (architecture, common, sdk) -- skip endpoint validation
- [ ] Files referencing internal endpoints -- flag for review
- [ ] Files with ASCII art diagrams -- preserve formatting exactly
- [ ] Files with image references -- verify images exist in `_static/`
- [ ] Files with `:ref:` cross-references -- verify anchors exist

---

## Validation Commands

### Sphinx Build
```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```
EXPECT: Zero errors, zero warnings

### AI-Native Compliance Check
```bash
# Check all overviews have AI Context
for f in bin-api-manager/docsdev/source/*_overview.rst; do
  grep -l "AI Context" "$f" > /dev/null || echo "MISSING AI Context: $f"
done

# Check all overviews have AI Implementation Hint
for f in bin-api-manager/docsdev/source/*_overview.rst; do
  grep -l "AI Implementation Hint" "$f" > /dev/null || echo "MISSING AI Hint: $f"
done
```
EXPECT: No output (all files compliant)

### FQDN URL Check
```bash
# Check for non-FQDN endpoint references
grep -n "POST /v1\|GET /v1\|PUT /v1\|DELETE /v1" bin-api-manager/docsdev/source/*_overview.rst
```
EXPECT: No matches (all use full URLs)

---

## Acceptance Criteria
- [ ] All 41 overview files audited against codebase sources of truth
- [ ] All documented endpoints verified against OpenAPI spec
- [ ] All status/enum values verified against model definitions
- [ ] All overview files have AI Context block
- [ ] All overview files have at least one AI Implementation Hint
- [ ] Sphinx HTML rebuild succeeds with zero errors
- [ ] RST source + built HTML committed together

## Completion Checklist
- [ ] Every overview file's endpoints match OpenAPI spec
- [ ] Every overview file's status values match model definitions
- [ ] No references to deprecated/removed features
- [ ] FQDN URLs used consistently
- [ ] No admin-only endpoints in user docs
- [ ] Cross-references resolve correctly
- [ ] HTML rebuild clean

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Large number of files causes context overflow | HIGH | Medium | Batch by resource domain, use subagents |
| Overview prose changes alter meaning incorrectly | MEDIUM | High | Minimal changes -- fix factual errors only, don't rewrite prose |
| Missing status values in code (defined at runtime) | LOW | Low | Check both const definitions and webhook.go for enum references |
| Sphinx build warnings from RST syntax | MEDIUM | Low | Clean rebuild after each batch |

## Notes
- Phase 1 struct audit revealed significant drift in conversation, conference, recording, and call resources -- expect corresponding overview drift
- Overview files are prose-heavy (avg ~400 lines) -- focus on factual accuracy of endpoints/status/enums, not prose style
- Some overview files (sdk, architecture, common, direct_hash) don't describe API resources -- audit for general accuracy only
- The `trunk_overview_trunking.rst` and `trunk_overview_domain_name.rst` are sub-files included by `trunk_overview.rst` -- audit all three together
